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
	vc *vmCtl
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
	return gre.l.Run(gre)
}

func (gre *gre) clean() {
	os.Remove(gre.socket)
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
	os.Remove(vc.outputFile.Name())
}

type vmInfo struct {
	VMErr     string
	Name      string
	ID        string
	Args      []string
	Stat      string // starting running exited
	StartTime time.Time
	EndTime   time.Time
}

type vmCtl struct {
	sync.Mutex
	vmInfo
	*tengo.VM
	vmErr      error // returned error when VM exits
	runMsg     *cmdRunMsg
	outputFile *os.File
}

type cmdPsMsg struct{}

type greVMInfo struct {
	Name    string
	VMInfos []*vmInfo
}

func (cmd cmdPsMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	gvi := &greVMInfo{Name: ggre.name, VMInfos: make([]*vmInfo, 0, len(ggre.vmids))}
	ggre.RLock()
	for i := len(ggre.vmids) - 1; i >= 0; i-- { // in reverse order
		vmid := ggre.vmids[i]
		vc := ggre.vms[vmid]
		gvi.VMInfos = append(gvi.VMInfos, &vc.vmInfo)
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
	vc.Stat = "starting"

	output, err := os.Create(logDir + vc.ID)
	if err != nil {
		greLogger.Errorln("cmdRunMsg: create output file error:", err)
		return nil, errors.New("output file not created")
	}
	vc.outputFile = output

	ggre.addVM(vc)

	if cmd.Interactive {
		conn.Send(vc.ID)
		session := conn.GetContext().(*session)
		session.vc = vc
		greLogger.Traceln("cmdRunMsg: sending redirectMsg")
		return redirectMsg{}, nil
	}

	go func() {
		defer output.Close()
		vm.In = null{}
		vm.Out = output
		greLogger.Traceln("cmdRunMsg: non-interactive")
		vc.StartTime = time.Now()
		vc.Stat = "running"
		err = vm.Run()
		vc.EndTime = time.Now()
		if err != nil {
			vc.vmErr = err
			vc.VMErr = err.Error()
			fmt.Fprintln(output, err)
		}
		vc.Stat = "exited"
		if cmd.AutoRemove {
			greLogger.Traceln("remove vm ctl")
			ggre.rmVM(vc)
		}
	}()

	return vc.ID, nil
}

type redirectAckMsg struct{}

func (redirectAckMsg) IsExclusive() {}
func (redirectAckMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	greLogger.Traceln("redirectAckMsg: enter")
	session := conn.GetContext().(*session)
	vc := session.vc
	defer vc.outputFile.Close()
	vm := vc.VM

	vm.In = conn.GetNetConn()
	output := io.MultiWriter(conn.GetNetConn(), vc.outputFile)
	vm.Out = output

	vc.StartTime = time.Now()
	vc.Stat = "running"
	err = vm.Run()
	vc.EndTime = time.Now()
	if err != nil {
		vc.vmErr = err
		vc.VMErr = err.Error()
		fmt.Fprintln(output, err)
	}

	vc.Stat = "exited"
	if vc.runMsg.AutoRemove {
		ggre.rmVM(vc)
	}

	greLogger.Traceln("redirectAckMsg: closing")
	return nil, io.EOF
}

func init() {
	gob.Register(redirectAckMsg{})
	gob.Register(&cmdRunMsg{})
	gob.Register(cmdPsMsg{})
	gob.Register(&greVMInfo{})
}
