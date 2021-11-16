package gshellos

import (
	"context"
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
	"github.com/godevsig/grepo/lib/sys/log"
	"github.com/traefik/yaegi/interp"
)

type grg struct {
	sync.RWMutex
	server *as.Server
	name   string
	lg     *log.Logger
	greids []string // keep the order
	gres   map[string]*greCtl
}

func (grg *grg) onNewStream(ctx as.Context) {
	ctx.SetContext(grg)
}

const greIDWidth = 6

func (grg *grg) addGRE(gc *greCtl) {
	grg.Lock()
	grg.greids = append(grg.greids, gc.ID)
	grg.gres[gc.ID] = gc
	grg.Unlock()
	grg.lg.Debugln("gre " + gc.ID + " added")
}

func (grg *grg) rmGRE(gc *greCtl) {
	grg.Lock()
	delete(grg.gres, gc.ID)
	greids := make([]string, 0, len(grg.greids)-1)
	for _, greid := range grg.greids {
		if greid != gc.ID {
			greids = append(greids, greid)
		}
	}
	grg.greids = greids
	grg.Unlock()
	os.Remove(gc.outputFile)
	grg.lg.Debugln("gre " + gc.ID + " removed")
}

const (
	greStatStarting int32 = iota
	greStatRunning
	greStatAborting
	greStatExited
)

var greStatString = []string{
	greStatStarting: "starting",
	greStatRunning:  "running",
	greStatAborting: "aborting",
	greStatExited:   "exited",
}

type greInfo struct {
	GREErr       string
	Name         string
	ID           string
	Args         []string
	Stat         string // starting running exited
	StartTime    time.Time
	EndTime      time.Time
	RestartedNum int
}

type greCtl struct {
	greInfo
	cancel     context.CancelFunc
	stdin      io.Reader
	stdout     io.Writer
	stderr     strings.Builder
	args       []string
	stat       int32
	greErr     error // returned error when GRE exits
	runMsg     *grgCmdRun
	outputFile string
	sh         *shell
}

func (gc *greCtl) newShell() error {
	src := string(gc.runMsg.ByteCode)
	src = strings.Replace(src, "main(", "_main(", 1)

	gc.sh = newShell(interp.Options{
		Stdin:  gc.stdin,
		Stdout: gc.stdout,
		Stderr: &gc.stderr,
		Args:   gc.args,
	})

	_, err := gc.sh.Eval(src)
	return err
}

func (gc *greCtl) runGRE() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc.cancel = cancel

	atomic.StoreInt32(&gc.stat, greStatRunning)
	gc.StartTime = time.Now()
	gc.greErr = nil
	gc.GREErr = ""
	gc.EndTime = time.Time{}

	if err := gc.newShell(); err != nil {
		fmt.Fprintln(&gc.stderr, err)
	} else {
		if _, err := gc.sh.EvalWithContext(ctx, "_main()"); err != nil {
			fmt.Fprintln(&gc.stderr, err)
			if p, ok := err.(interp.Panic); ok {
				fmt.Fprintln(&gc.stderr, string(p.Stack))
			}
		}
	}

	gc.EndTime = time.Now()
	if len(gc.stderr.String()) != 0 {
		gc.greErr = fmt.Errorf("%s", gc.stderr.String())
		gc.GREErr = gc.stderr.String()
	}
	atomic.StoreInt32(&gc.stat, greStatExited)
}

type grgGREInfo struct {
	Name     string
	GREInfos []*greInfo
}

type grgCmdRun struct {
	File        string
	Args        []string
	Interactive bool
	AutoRemove  bool
	ByteCode    []byte
}

func (msg *grgCmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)
	grg.lg.Debugf("grgCmdRun: file: %v, args: %v, interactive: %v\n", msg.File, msg.Args, msg.Interactive)

	name := filepath.Base(msg.File)
	gc := &greCtl{args: msg.Args, runMsg: msg}
	gc.Name = strings.TrimSuffix(name, filepath.Ext(name))
	gc.ID = genID(greIDWidth)
	gc.greInfo.Args = msg.Args
	atomic.StoreInt32(&gc.stat, greStatStarting)

	gc.outputFile = workDir + "/logs/" + gc.ID
	output, err := os.Create(gc.outputFile)
	if err != nil {
		grg.lg.Errorln("grgCmdRun: create output file error:", err)
		return errors.New("output file not created")
	}
	grg.addGRE(gc)

	if msg.Interactive {
		grg.lg.Debugln("grgCmdRun: interactive")
		defer output.Close()
		clientIO := as.NewStreamIO(stream)
		gc.stdin = clientIO
		gc.stdout = multiWriter(clientIO, output)
		gc.runGRE()
		if msg.AutoRemove {
			grg.rmGRE(gc)
		}
		clientIO.Close()
		return nil
	}

	go func() {
		grg.lg.Debugln("grgCmdRun: non-interactive")
		defer output.Close()
		gc.stdin = null{}
		gc.stdout = output
		gc.runGRE()
		if msg.AutoRemove {
			grg.rmGRE(gc)
		}
	}()

	return gc.ID
}

type grgCmdQuery struct {
	IDPatten []string
}

func (msg *grgCmdQuery) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)
	ggi := &grgGREInfo{Name: grg.name}
	pattenStr := ""
	if len(msg.IDPatten) == 0 { // list all
		ggi.GREInfos = make([]*greInfo, 0, len(grg.greids))
	} else {
		pattenStr = "^" + strings.Join(msg.IDPatten, "$ ^") + "$"
	}

	grg.RLock()
	for i := len(grg.greids) - 1; i >= 0; i-- { // in reverse order
		greid := grg.greids[i]
		gc := grg.gres[greid]
		if len(pattenStr) == 0 || // match all
			strings.Contains(pattenStr, "^"+greid+"$") || // match greid
			strings.Contains(pattenStr, "^"+gc.Name+"$") { // match name
			gc.Stat = greStatString[gc.stat]
			ggi.GREInfos = append(ggi.GREInfos, &gc.greInfo)
		}
	}
	grg.RUnlock()
	return ggi
}

type grgCmdPatternAction struct {
	IDPattern []string
	Cmd       string
}

func (msg *grgCmdPatternAction) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)
	pattenStr := "^" + strings.Join(msg.IDPattern, "$ ^") + "$"
	var gcs []*greCtl
	grg.RLock()
	for greid, gc := range grg.gres {
		if strings.Contains(pattenStr, "^"+greid+"$") || // match greid
			strings.Contains(pattenStr, "^"+gc.Name+"$") { // match name
			gcs = append(gcs, gc)
		}
	}
	grg.RUnlock()

	var ids []string
	for _, gc := range gcs {
		switch msg.Cmd {
		case "stop":
			if gc.stat == greStatRunning { // no need to atomic
				if _, err := gc.sh.Eval("Stop()"); err != nil { // try to call Stop() if there is one
					gc.cancel()
				}
				atomic.CompareAndSwapInt32(&gc.stat, greStatRunning, greStatAborting)
				ids = append(ids, gc.ID)
			}
		case "rm":
			if gc.stat == greStatExited {
				grg.rmGRE(gc)
				ids = append(ids, gc.ID)
			}
		case "restart":
			if gc.stat == greStatExited {
				gc.stdin = null{}
				logFile, err := os.Create(gc.outputFile)
				if err != nil {
					grg.lg.Errorf("output file not created: %v", err)
					break
				}
				gc.stdout = logFile
				gc := gc
				go func() {
					defer logFile.Close()
					gc.RestartedNum++
					gc.runGRE()
				}()
				ids = append(ids, gc.ID)
			}
		}
	}

	if msg.Cmd == "rm" && len(grg.greids) == 0 {
		grg.server.Close()
	}

	return ids
}

type processInfo struct {
	grgName    string
	pid        int
	runningCnt int
}

// reply with processInfo
type getProcessInfo struct{}

func (msg getProcessInfo) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)
	pid := os.Getpid()
	runningCnt := 0
	grg.RLock()
	for _, gc := range grg.gres {
		if gc.stat == greStatRunning {
			runningCnt++
		}
	}
	grg.RUnlock()
	return processInfo{grg.name, pid, runningCnt}
}

var grgKnownMsgs = []as.KnownMessage{
	(*grgCmdRun)(nil),
	(*grgCmdQuery)(nil),
	(*grgCmdPatternAction)(nil),
	getProcessInfo{},
}

func init() {
	as.RegisterType((*grgCmdRun)(nil))
	as.RegisterType((*grgCmdQuery)(nil))
	as.RegisterType((*grgGREInfo)(nil))
	as.RegisterType((*grgCmdPatternAction)(nil))
	as.RegisterType(getProcessInfo{})
	as.RegisterType(processInfo{})
}
