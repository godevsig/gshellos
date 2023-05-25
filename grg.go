package gshellos

import (
	"context"
	"encoding/gob"
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

type processInfo struct {
	name       string
	rtPriority int
	maxProcs   int
	statDir    string
	pid        int
	killing    bool
}

type grg struct {
	sync.RWMutex
	processInfo
	workDir string
	server  *as.Server
	lg      *log.Logger
	greids  []string // keep the order
	gres    map[string]*greCtl
}

func (grg *grg) onNewStream(ctx as.Context) {
	ctx.SetContext(grg)
}

const greIDWidth = 6

func (grg *grg) loadGREs() error {
	gres, err := filepath.Glob(grg.statDir + "/*")
	if err != nil {
		return err
	}

	for _, greStatDir := range gres {
		func() {
			greid := filepath.Base(greStatDir)
			if len(greid) == 0 || greid == ".lock" {
				return
			}
			fgi, err := os.Open(greStatDir + "/greInfo")
			if err != nil {
				grg.lg.Warnln(err)
				return
			}
			defer fgi.Close()
			gi := &greInfo{}
			if err := gob.NewDecoder(fgi).Decode(gi); err != nil {
				grg.lg.Warnf("decode greInfo for %s failed", greid)
				return
			}
			frm, err := os.Open(greStatDir + "/runMsg")
			if err != nil {
				grg.lg.Warnln(err)
				return
			}
			defer frm.Close()
			runMsg := &grgCmdRun{}
			if err := gob.NewDecoder(frm).Decode(runMsg); err != nil {
				grg.lg.Warnf("decode runMsg for %s failed", greid)
				return
			}
			gc, err := grg.newGRE(gi, runMsg)
			if err != nil {
				grg.lg.Errorln(err)
				return
			}
			grg.addGRE(gc)

			switch gi.Stat {
			case "starting", "running":
				go grg.runGRE(gc)
				grg.lg.Infof("gre %s restarted", greid)
			case "aborting", "exited":
				gc.changeStat(greStatExited)
			default:
				grg.lg.Infof("gre %s not restarted with status %s", greid, gi.Stat)
			}
		}()
	}
	return nil
}

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
	gc.close()
	grg.lg.Debugln("gre " + gc.ID + " removed")

	grg.Lock()
	if len(grg.greids) == 0 {
		grg.server.Close()
	}
	grg.Unlock()
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
	GREErr             string
	Name               string
	ID                 string
	Args               []string
	Stat               string // starting running exited
	StartTime          time.Time
	EndTime            time.Time
	RestartedNum       int
	AutoRestartBalance uint // the remaining number of auto restart
}

type greCtl struct {
	*greInfo
	cancel     context.CancelFunc
	log        *os.File
	stdin      io.Reader
	stdout     io.Writer
	stderr     strings.Builder
	args       []string
	stat       int32
	greErr     error // returned error when GRE exits
	runMsg     *grgCmdRun
	outputFile string
	statDir    string
	gsh        *gshell
	codeDir    string
}

// gi is not nil when loading from file
func (grg *grg) newGRE(gi *greInfo, runMsg *grgCmdRun) (*greCtl, error) {
	gc := &greCtl{args: runMsg.Args, runMsg: runMsg}
	gc.greInfo = gi
	if gi == nil {
		gc.greInfo = &greInfo{}
		name := filepath.Base(runMsg.Args[0])
		gc.Name = strings.TrimSuffix(name, filepath.Ext(name))
		gc.ID = genID(greIDWidth)
		gc.greInfo.Args = runMsg.Args
		gc.RestartedNum = 0
		gc.AutoRestartBalance = runMsg.AutoRestartMax
	}
	gc.outputFile = filepath.Join(grg.workDir, "logs", gc.ID)
	gc.statDir = filepath.Join(grg.statDir, gc.ID)

	if gi == nil {
		if err := os.MkdirAll(gc.statDir, 0755); err != nil {
			return nil, err
		}
		if err := gc.reset(); err != nil {
			return nil, err
		}
		if err := gc.runMsgToFile(); err != nil {
			return nil, err
		}
	}

	tmpDir, err := os.MkdirTemp(gshellTempDir, "gre-code-")
	if err != nil {
		return nil, err
	}
	gc.codeDir = tmpDir

	if err := unzipBufferToPath(runMsg.CodeZip, tmpDir); err != nil {
		return nil, err
	}
	runMsg.CodeZip = nil // release the mem sooner

	return gc, nil
}

func (grg *grg) runGRE(gc *greCtl) {
	for {
		gc.runGRE()
		if gc.AutoRestartBalance == 0 {
			break
		}
		if err := gc.reset(); err != nil {
			break
		}
		gc.RestartedNum++
	}
	if gc.runMsg.AutoRemove {
		grg.rmGRE(gc)
	}
}

func (gc *greCtl) runMsgToFile() error {
	f, err := os.Create(gc.statDir + "/runMsg")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(gc.runMsg); err != nil {
		return err
	}
	return nil
}

func (gc *greCtl) greInfoToFile() error {
	f, err := os.Create(gc.statDir + "/greInfo")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(gc.greInfo); err != nil {
		return err
	}
	return nil
}

func (gc *greCtl) changeStat(newStat int32) {
	atomic.StoreInt32(&gc.stat, newStat)
	gc.Stat = greStatString[gc.stat]
}

func (gc *greCtl) changeStatIf(oldStat, newStat int32) {
	if atomic.CompareAndSwapInt32(&gc.stat, oldStat, newStat) {
		gc.Stat = greStatString[gc.stat]
	}
}

func (gc *greCtl) reset() error {
	gc.changeStat(greStatStarting)
	output, err := os.OpenFile(gc.outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("GRE output file not created: %v", err)
	}
	gc.log = output
	gc.stdin = nullIO{}
	gc.stdout = output
	return nil
}

func (gc *greCtl) close() {
	if gc.gsh != nil {
		gc.gsh.close()
	}
	os.Remove(gc.outputFile)
	os.RemoveAll(gc.statDir)
	os.RemoveAll(gc.codeDir)
}

func (gc *greCtl) newShell() (err error) {
	gc.gsh, err = newShell(interp.Options{
		Stdin:  gc.stdin,
		Stdout: gc.stdout,
		Stderr: &gc.stderr,
		Args:   gc.args,
	})
	return
}

func (gc *greCtl) runGRE() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc.cancel = cancel

	gc.StartTime = time.Now()
	gc.greErr = nil
	gc.GREErr = ""
	gc.EndTime = time.Time{}

	gc.changeStat(greStatRunning)
	gc.greInfoToFile()

	if err := gc.newShell(); err != nil {
		fmt.Fprintln(&gc.stderr, err)
	} else {
		if err := gc.gsh.evalPathWithContext(ctx, gc.codeDir); err != nil {
			fmt.Fprintln(&gc.stderr, err)
			if p, ok := err.(interp.Panic); ok {
				fmt.Fprintln(&gc.stderr, string(p.Stack))
			}
		}
	}

	gc.EndTime = time.Now()
	stderrStr := gc.stderr.String()
	if len(stderrStr) != 0 {
		gc.greErr = fmt.Errorf("%s", stderrStr)
		gc.GREErr = stderrStr
		fmt.Fprint(gc.stdout, stderrStr)
	}
	gc.log.Close()
	if gc.greErr == nil {
		gc.AutoRestartBalance = 0
	}
	if gc.AutoRestartBalance > 0 {
		gc.AutoRestartBalance--
	}
	gc.changeStat(greStatExited)
	gc.greInfoToFile()
}

type grgGREInfo struct {
	Name     string
	GREInfos []*greInfo
}

// JobCmd is the job in grgCmdRun
type JobCmd struct {
	Args           []string `yaml:"args,omitempty"`
	AutoRemove     bool     `yaml:"auto-remove,omitempty"`
	AutoRestartMax uint     `yaml:"auto-restart-max,omitempty"` // user defined max auto restart count
	CodeZip        []byte   `yaml:"code-zip,omitempty"`
}

// JobInfo is the job in joblist
type JobInfo struct {
	Cmd           string `yaml:"cmd"`
	JobCmd        `yaml:",inline"`
	CodeZipBase64 string `yaml:"code-zip-base64,omitempty"`
}

type grgCmdRun struct {
	JobCmd
	Interactive bool
	AutoImport  bool
}

func (msg *grgCmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)
	grg.lg.Debugf("grgCmdRun: args: %v, interactive: %v\n", msg.Args, msg.Interactive)

	if msg.CodeZip == nil {
		filePath := msg.Args[0]
		c := as.NewClient(as.WithLogger(grg.lg)).SetDiscoverTimeout(0)
		conn := <-c.Discover(godevsigPublisher, "codeRepo")
		if conn == nil {
			return fmt.Errorf("%s_codeRepo service not running", godevsigPublisher)
		}
		defer conn.Close()

		var zip []byte
		if err := conn.SendRecv(getCode{filePath, msg.AutoImport}, &zip); err != nil {
			return err
		}
		msg.CodeZip = zip
	}

	gc, err := grg.newGRE(nil, msg)
	if err != nil {
		grg.lg.Errorln(err)
		return err
	}
	grg.addGRE(gc)

	if msg.Interactive {
		grg.lg.Debugln("grgCmdRun: interactive")
		clientIO := as.NewStreamIO(stream)
		defer clientIO.Close()
		gc.stdin = clientIO
		gc.stdout = multiWriter(clientIO, gc.log)
		gc.runGRE()
		if msg.AutoRemove {
			grg.rmGRE(gc)
		}
		return nil
	}

	go grg.runGRE(gc)
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
			ggi.GREInfos = append(ggi.GREInfos, gc.greInfo)
		}
	}
	grg.RUnlock()
	return ggi
}

type grgJoblist struct {
	Name       string
	RtPriority int `yaml:"rt-priority,omitempty"`
	Maxprocs   int `yaml:"max-procs,omitempty"`
	Jobs       []*JobInfo
}

// reply grgJoblist{}
type grgCmdJoblist struct{}

func (msg grgCmdJoblist) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)

	grgjl := &grgJoblist{
		Name:       grg.name,
		RtPriority: grg.rtPriority,
		Maxprocs:   grg.maxProcs,
	}

	grg.RLock()
	for _, gc := range grg.gres {
		ji := &JobInfo{JobCmd: gc.runMsg.JobCmd}
		grgjl.Jobs = append(grgjl.Jobs, ji)
	}
	grg.RUnlock()

	return grgjl
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
				gc.cancel()
				gc.changeStatIf(greStatRunning, greStatAborting)
				ids = append(ids, gc.ID)
			}
		case "rm":
			if gc.stat == greStatExited {
				grg.rmGRE(gc)
				ids = append(ids, gc.ID)
			}
		case "start":
			if gc.stat == greStatExited {
				if err := gc.reset(); err != nil {
					grg.lg.Errorln(err)
					break
				}
				gc := gc
				go gc.runGRE()
				ids = append(ids, gc.ID)
			}
		}
	}

	return ids
}

// reply with &processInfo
type grgCmdKill struct{}

func (msg grgCmdKill) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)

	allExited := func() bool {
		allExited := true
		grg.RLock()
		defer grg.RUnlock()
		for _, gc := range grg.gres {
			if gc.stat != greStatExited {
				allExited = false
				continue
			}
			grg.RUnlock()
			grg.rmGRE(gc)
			grg.RLock()
		}
		return allExited
	}

	if allExited() {
		grg.lg.Infoln("command kill received and all jobs done, closing")
		// Is below needed? Closing before sending reply seems possible?
		// time.AfterFunc(time.Second*3, func() { grg.server.Close() })
		grg.server.Close()
		grg.killing = true
	}
	return &grg.processInfo
}

var grgKnownMsgs = []as.KnownMessage{
	(*grgCmdRun)(nil),
	(*grgCmdQuery)(nil),
	grgCmdJoblist{},
	(*grgCmdPatternAction)(nil),
	grgCmdKill{},
}

func init() {
	as.RegisterType((*grgCmdRun)(nil))
	as.RegisterType((*grgCmdQuery)(nil))
	as.RegisterType((*grgGREInfo)(nil))
	as.RegisterType(grgCmdJoblist{})
	as.RegisterType((*grgJoblist)(nil))
	as.RegisterType((*grgCmdPatternAction)(nil))
	as.RegisterType(grgCmdKill{})
	as.RegisterType((*processInfo)(nil))
}
