package gshellos

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	logDir    = workDir + "logs/"
	greLogger *log.Logger
	greStream = log.NewStream("gre")
	ggre      *gre // a process can only have one gre instance by design
)

type null struct{}

func (null) Close() error                  { return nil }
func (null) Write(buf []byte) (int, error) { return len(buf), nil }
func (null) Read(buf []byte) (int, error)  { return 0, io.EOF }

func init() {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(err)
	}
	// same file shared by multiple gshell processes in append mode.
	greStream.SetOutput("file:" + workDir + "gre.log")
}

type gre struct {
	sync.RWMutex
	name   string
	socket string
	l      *sm.Listener
	vmids  []string // keep the order
	vms    map[string]*vmCtl
}

type session struct {
	var1 string
	var2 int
}

func newgre(name string) (*gre, error) {
	if greLogger = greStream.GetLogger(name); greLogger == nil {
		greLogger = greStream.NewLogger(name, log.Linfo)
	}

	gre := &gre{
		name:   name,
		socket: workDir + "gre-" + name + ".sock",
		vms:    make(map[string]*vmCtl),
	}
	l, err := sm.Listen("unix", gre.socket, sm.WithLogger(greLogger))
	if err != nil {
		return nil, err
	}
	gre.l = l
	ggre = gre
	return gre, nil
}

func (gre *gre) run() error {
	defer os.Remove(gre.socket)
	return gre.l.Run(gre)
}

func (gre *gre) close() {
	gre.l.Close()
}

func rungre(name string) error {
	gre, err := newgre(name)
	if err != nil {
		return err
	}
	return gre.run()
}

func (gre *gre) OnConnect(conn sm.Conn) error {
	greLogger.Debugln("new connection", conn.GetNetConn().RemoteAddr().String())
	ctx := &session{}
	conn.SetContext(ctx)
	return nil
}

type getProcess struct{}

func (req getProcess) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	pid := os.Getpid()
	return pid, nil
}

func genVMID() string {
	b := make([]byte, 6)
	rand.Read(b)
	id := hex.EncodeToString(b)
	return id
}

func (gre *gre) addVM(vc *vmCtl) {
	gre.Lock()
	gre.vmids = append(gre.vmids, vc.ID)
	gre.vms[vc.ID] = vc
	gre.Unlock()
	greLogger.Traceln("vm " + vc.ID + " added")
}

func (gre *gre) rmVM(vc *vmCtl) {
	gre.Lock()
	delete(gre.vms, vc.ID)
	vmids := make([]string, 0, len(gre.vmids)-1)
	for _, vmid := range gre.vmids {
		if vmid != vc.ID {
			vmids = append(vmids, vmid)
		}
	}
	gre.vmids = vmids
	gre.Unlock()
	os.Remove(vc.outputFile)
	greLogger.Traceln("vm " + vc.ID + " removed")
}

const (
	vmStatStarting int32 = iota
	vmStatRunning
	vmStatAborting
	vmStatExited
)

var vmStatString = []string{
	vmStatStarting: "starting",
	vmStatRunning:  "running",
	vmStatAborting: "aborting",
	vmStatExited:   "exited",
}

type vmInfo struct {
	VMErr        string
	Name         string
	ID           string
	Args         []string
	Stat         string // starting running exited
	StartTime    time.Time
	EndTime      time.Time
	RestartedNum int
}

type vmCtl struct {
	//sync.Mutex
	vmInfo
	*tengo.VM
	stat       int32
	vmErr      error // returned error when VM exits
	runMsg     *cmdRunMsg
	outputFile string
}

func (vc *vmCtl) runVM() {
	vm := vc.VM
	//vc.Stat = "running"
	atomic.StoreInt32(&vc.stat, vmStatRunning)
	vc.StartTime = time.Now()
	err := vm.Run()
	vc.EndTime = time.Now()
	if err != nil {
		vc.vmErr = err
		vc.VMErr = err.Error()
		fmt.Fprintln(vm.Out, err)
	}
	//vc.Stat = "exited"
	atomic.StoreInt32(&vc.stat, vmStatExited)
}

type greVMInfo struct {
	Name    string
	VMInfos []*vmInfo
}

type cmdPatternActionMsg struct {
	IDPattern []string
	Cmd       string
}

func (cmd cmdPatternActionMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	pattenStr := "^" + strings.Join(cmd.IDPattern, "$ ^") + "$"
	var vcs []*vmCtl
	ggre.RLock()
	for vmid, vc := range ggre.vms {
		if strings.Contains(pattenStr, "^"+vmid+"$") || // match vmid
			strings.Contains(pattenStr, "^"+vc.Name+"$") { // match name
			vcs = append(vcs, vc)
		}
	}
	ggre.RUnlock()

	var ids []string
	for _, vc := range vcs {
		switch cmd.Cmd {
		case "kill":
			if vc.stat == vmStatRunning { // no need to atomic
				vc.VM.Abort()
				//vc.Stat = "aborting"
				atomic.CompareAndSwapInt32(&vc.stat, vmStatRunning, vmStatAborting)
				ids = append(ids, vc.ID)
			}
		case "rm":
			if vc.stat == vmStatExited {
				ggre.rmVM(vc)
				ids = append(ids, vc.ID)
			}
		case "restart":
			if vc.stat == vmStatExited {
				vm := vc.VM
				vm.In = null{}
				logFile, err := os.Create(vc.outputFile)
				if err != nil {
					greLogger.Errorln(errorHere(err))
					break
				}
				vm.Out = logFile
				vc := vc
				go func() {
					defer logFile.Close()
					vc.RestartedNum++
					vc.runVM()
				}()
				ids = append(ids, vc.ID)
			}
		}
	}

	return ids, nil
}

type cmdQueryMsg struct {
	IDPatten []string
}

func (cmd cmdQueryMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	gvi := &greVMInfo{Name: ggre.name}
	pattenStr := ""
	if len(cmd.IDPatten) == 0 { // list all
		gvi.VMInfos = make([]*vmInfo, 0, len(ggre.vmids))
	} else {
		pattenStr = "^" + strings.Join(cmd.IDPatten, "$ ^") + "$"
	}

	ggre.RLock()
	for i := len(ggre.vmids) - 1; i >= 0; i-- { // in reverse order
		vmid := ggre.vmids[i]
		vc := ggre.vms[vmid]
		if len(pattenStr) == 0 || // match all
			strings.Contains(pattenStr, "^"+vmid+"$") || // match vmid
			strings.Contains(pattenStr, "^"+vc.Name+"$") { // match name
			vc.Stat = vmStatString[vc.stat]
			gvi.VMInfos = append(gvi.VMInfos, &vc.vmInfo)
		}
	}
	ggre.RUnlock()
	return gvi, nil
}

type cmdRunMsg struct {
	File        string
	Args        []string
	Interactive bool
	AutoRemove  bool
	ByteCode    []byte
}

func (cmd *cmdRunMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	greLogger.Debugf("cmdRunMsg: file: %v, args: %v, interactive: %v\n", cmd.File, cmd.Args, cmd.Interactive)
	sh := newShell()
	bytecode := &tengo.Bytecode{}
	err := bytecode.Decode(bytes.NewReader(cmd.ByteCode), sh.modules)
	//err = errors.New("test") // for test
	if err != nil {
		greLogger.Errorf("cmdRunMsg: decode %s error: %v", cmd.File, err)
		return nil, errors.New("bad bytecode")
	}

	name := filepath.Base(cmd.File)
	vm := tengo.NewVM(bytecode, sh.globals, -1)
	vm.Args = cmd.Args
	vc := &vmCtl{
		VM:     vm,
		runMsg: cmd,
	}
	vc.Name = strings.TrimSuffix(name, filepath.Ext(name))
	vc.ID = genVMID()
	vc.vmInfo.Args = cmd.Args
	//vc.Stat = "starting"
	atomic.StoreInt32(&vc.stat, vmStatStarting)

	vc.outputFile = logDir + vc.ID
	output, err := os.Create(vc.outputFile)
	if err != nil {
		greLogger.Errorln("cmdRunMsg: create output file error:", err)
		return nil, errors.New("output file not created")
	}
	ggre.addVM(vc)

	if cmd.Interactive {
		greLogger.Traceln("cmdRunMsg: interactive")
		defer output.Close()
		vm.In = conn.GetNetConn()
		vm.Out = multiWriter(conn.GetNetConn(), output)
		vc.runVM()
		if cmd.AutoRemove {
			ggre.rmVM(vc)
		}
		return nil, io.EOF
	}

	go func() {
		greLogger.Traceln("cmdRunMsg: non-interactive")
		defer output.Close()
		vm.In = null{}
		vm.Out = output
		vc.runVM()
		if cmd.AutoRemove {
			ggre.rmVM(vc)
		}
	}()

	return vc.ID, nil
}

func init() {
	gob.Register(&cmdRunMsg{})
	gob.Register(cmdQueryMsg{})
	gob.Register(cmdPatternActionMsg{})
	gob.Register(&greVMInfo{})
	gob.Register(getProcess{})
}
