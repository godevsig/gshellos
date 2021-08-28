package gshellos

import (
	"bytes"
	"crypto/rand"
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
	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/gshellos/log"
)

type gre struct {
	sync.RWMutex
	name  string
	lg    *log.Logger
	vmids []string // keep the order
	vms   map[string]*vmCtl
}

func (gre *gre) onNewStream(ctx as.Context) {
	ctx.SetContext(gre)
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
	gre.lg.Debugln("vm " + vc.ID + " added")
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
	gre.lg.Debugln("vm " + vc.ID + " removed")
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
	vmInfo
	*tengo.VM
	stat       int32
	vmErr      error // returned error when VM exits
	runMsg     *greCmdRun
	outputFile string
}

func (vc *vmCtl) runVM() {
	vm := vc.VM
	atomic.StoreInt32(&vc.stat, vmStatRunning)
	vc.StartTime = time.Now()
	err := vm.Run()
	vc.EndTime = time.Now()
	if err != nil {
		vc.vmErr = err
		vc.VMErr = err.Error()
		fmt.Fprintln(vm.Out, err)
	}
	atomic.StoreInt32(&vc.stat, vmStatExited)
}

type greVMInfo struct {
	Name    string
	VMInfos []*vmInfo
}

type greCmdRun struct {
	File        string
	Args        []string
	Interactive bool
	AutoRemove  bool
	ByteCode    []byte
}

func (msg *greCmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	gre := stream.GetContext().(*gre)
	gre.lg.Debugf("greCmdRun: file: %v, args: %v, interactive: %v\n", msg.File, msg.Args, msg.Interactive)

	sh := newShell()
	bytecode := &tengo.Bytecode{}
	err := bytecode.Decode(bytes.NewReader(msg.ByteCode), sh.modules)
	if err != nil {
		gre.lg.Errorf("greCmdRun: decode %s error: %v", msg.File, err)
		return errors.New("bad bytecode")
	}

	name := filepath.Base(msg.File)
	vm := tengo.NewVM(bytecode, sh.globals, -1)
	vm.Args = msg.Args
	vc := &vmCtl{
		VM:     vm,
		runMsg: msg,
	}
	vc.Name = strings.TrimSuffix(name, filepath.Ext(name))
	vc.ID = genVMID()
	vc.vmInfo.Args = msg.Args
	atomic.StoreInt32(&vc.stat, vmStatStarting)

	vc.outputFile = logDir + vc.ID
	output, err := os.Create(vc.outputFile)
	if err != nil {
		gre.lg.Errorln("greCmdRun: create output file error:", err)
		return errors.New("output file not created")
	}
	gre.addVM(vc)

	if msg.Interactive {
		gre.lg.Debugln("greCmdRun: interactive")
		defer output.Close()
		vm.In = stream
		vm.Out = multiWriter(stream, output)
		vc.runVM()
		if msg.AutoRemove {
			gre.rmVM(vc)
		}
		return io.EOF
	}

	go func() {
		gre.lg.Debugln("greCmdRun: non-interactive")
		defer output.Close()
		vm.In = null{}
		vm.Out = output
		vc.runVM()
		if msg.AutoRemove {
			gre.rmVM(vc)
		}
	}()

	return vc.ID
}

type greCmdQuery struct {
	IDPatten []string
}

func (msg *greCmdQuery) Handle(stream as.ContextStream) (reply interface{}) {
	gre := stream.GetContext().(*gre)
	gvi := &greVMInfo{Name: gre.name}
	pattenStr := ""
	if len(msg.IDPatten) == 0 { // list all
		gvi.VMInfos = make([]*vmInfo, 0, len(gre.vmids))
	} else {
		pattenStr = "^" + strings.Join(msg.IDPatten, "$ ^") + "$"
	}

	gre.RLock()
	for i := len(gre.vmids) - 1; i >= 0; i-- { // in reverse order
		vmid := gre.vmids[i]
		vc := gre.vms[vmid]
		if len(pattenStr) == 0 || // match all
			strings.Contains(pattenStr, "^"+vmid+"$") || // match vmid
			strings.Contains(pattenStr, "^"+vc.Name+"$") { // match name
			vc.Stat = vmStatString[vc.stat]
			gvi.VMInfos = append(gvi.VMInfos, &vc.vmInfo)
		}
	}
	gre.RUnlock()
	return gvi
}

type greCmdPatternAction struct {
	IDPattern []string
	Cmd       string
}

func (msg *greCmdPatternAction) Handle(stream as.ContextStream) (reply interface{}) {
	gre := stream.GetContext().(*gre)
	pattenStr := "^" + strings.Join(msg.IDPattern, "$ ^") + "$"
	var vcs []*vmCtl
	gre.RLock()
	for vmid, vc := range gre.vms {
		if strings.Contains(pattenStr, "^"+vmid+"$") || // match vmid
			strings.Contains(pattenStr, "^"+vc.Name+"$") { // match name
			vcs = append(vcs, vc)
		}
	}
	gre.RUnlock()

	var ids []string
	for _, vc := range vcs {
		switch msg.Cmd {
		case "kill":
			if vc.stat == vmStatRunning { // no need to atomic
				vc.VM.Abort()
				atomic.CompareAndSwapInt32(&vc.stat, vmStatRunning, vmStatAborting)
				ids = append(ids, vc.ID)
			}
		case "rm":
			if vc.stat == vmStatExited {
				gre.rmVM(vc)
				ids = append(ids, vc.ID)
			}
		case "restart":
			if vc.stat == vmStatExited {
				vm := vc.VM
				vm.In = null{}
				logFile, err := os.Create(vc.outputFile)
				if err != nil {
					gre.lg.Errorf("output file not created: %v", err)
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

	return ids
}

type getProcess struct{}

func (msg getProcess) Handle(stream as.ContextStream) (reply interface{}) {
	pid := os.Getpid()
	return pid
}

var greKnownMsgs = []as.KnownMessage{
	(*greCmdRun)(nil),
	(*greCmdQuery)(nil),
	(*greCmdPatternAction)(nil),
	getProcess{},
}

func init() {
	as.RegisterType((*greCmdRun)(nil))
	as.RegisterType((*greCmdQuery)(nil))
	as.RegisterType((*greVMInfo)(nil))
	as.RegisterType((*greCmdPatternAction)(nil))
	as.RegisterType(getProcess{})
}
