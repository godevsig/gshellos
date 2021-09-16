package gshellos

import (
	"context"
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

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/gshellos/log"
	"github.com/traefik/yaegi/interp"
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
	cancel     context.CancelFunc
	stdin      io.Reader
	stdout     io.Writer
	stderr     strings.Builder
	args       []string
	stat       int32
	vmErr      error // returned error when VM exits
	runMsg     *greCmdRun
	outputFile string
	sh         *shell
}

func (vc *vmCtl) newShell() error {
	src := string(vc.runMsg.ByteCode)
	if len(src) != 0 {
		src = strings.Replace(src, "main(", "_main(", 1)
	} else {
		pkg := vc.runMsg.File
		pkgBase := filepath.Base(pkg)
		src = fmt.Sprintf(`
package main
import (
	"fmt"
	"os"
	"%s"
)
func Stop() { %s.Stop() }
func _main() {
	if err := %s.Start(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}`, pkg, pkgBase, pkgBase)
	}

	vc.sh = newShell(interp.Options{
		Stdin:  vc.stdin,
		Stdout: vc.stdout,
		Stderr: &vc.stderr,
		Args:   vc.args,
	})

	_, err := vc.sh.Eval(src)
	return err
}

func (vc *vmCtl) runVM() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vc.cancel = cancel

	atomic.StoreInt32(&vc.stat, vmStatRunning)
	vc.StartTime = time.Now()

	if err := vc.newShell(); err != nil {
		fmt.Fprintln(&vc.stderr, err)
	} else {
		if _, err := vc.sh.EvalWithContext(ctx, "_main()"); err != nil {
			fmt.Fprintln(&vc.stderr, err)
		}
	}

	vc.EndTime = time.Now()
	if len(vc.stderr.String()) != 0 {
		vc.vmErr = fmt.Errorf("%s", vc.stderr.String())
		vc.VMErr = vc.stderr.String()
		fmt.Fprintln(vc.stdout, vc.VMErr)
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

	name := filepath.Base(msg.File)
	upper := filepath.Base(filepath.Dir(msg.File))
	if upper != "." {
		name = upper + "/" + name
	}

	vc := &vmCtl{args: msg.Args, runMsg: msg}
	vc.Name = strings.TrimSuffix(name, filepath.Ext(name))
	vc.ID = genVMID()
	vc.vmInfo.Args = msg.Args
	atomic.StoreInt32(&vc.stat, vmStatStarting)

	vc.outputFile = workDir + "/logs/" + vc.ID
	output, err := os.Create(vc.outputFile)
	if err != nil {
		gre.lg.Errorln("greCmdRun: create output file error:", err)
		return errors.New("output file not created")
	}
	gre.addVM(vc)

	if msg.Interactive {
		gre.lg.Debugln("greCmdRun: interactive")
		defer output.Close()
		vc.stdin = stream
		vc.stdout = multiWriter(stream, output)
		vc.runVM()
		if msg.AutoRemove {
			gre.rmVM(vc)
		}
		return io.EOF
	}

	go func() {
		gre.lg.Debugln("greCmdRun: non-interactive")
		defer output.Close()
		vc.stdin = null{}
		vc.stdout = output
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
				vc.sh.Eval("Stop()") // try to call Stop() if there is one
				vc.cancel()
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
				vc.stdin = null{}
				logFile, err := os.Create(vc.outputFile)
				if err != nil {
					gre.lg.Errorf("output file not created: %v", err)
					break
				}
				vc.stdout = logFile
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
