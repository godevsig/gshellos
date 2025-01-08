package gshellos

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
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
			switch gc.Stat {
			case greStatInit, greStatRunning:
				go grg.runGRE(gc)
				grg.lg.Infof("gre %s restarted", greid)
			case greStatAborting, greStatCancelling:
				gc.changeStat(greStatAborted)
			default:
				grg.lg.Infof("gre %s not restarted with status %s", greid, greStatToStr(gi.Stat))
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

type greInfo struct {
	GREErr             string
	Name               string
	ID                 string
	Args               []string
	Stat               greStat
	StartTime          time.Time
	EndTime            time.Time
	RestartedNum       int
	AutoRestartBalance uint   // the remaining number of auto restart
	RequestedBy        string // by which provider ID
}

type greCtl struct {
	*greInfo
	cancel     context.CancelFunc
	stdin      io.Reader
	stdout     io.Writer
	stderr     *strings.Builder
	greErr     error // returned error when GRE exits
	runMsg     *grgCmdRun
	outputFile string
	statDir    string
	gsh        *gshell
	codeDir    string
}

// gi is not nil when loading from file
func (grg *grg) newGRE(gi *greInfo, runMsg *grgCmdRun) (*greCtl, error) {
	gc := &greCtl{runMsg: runMsg}
	if gi == nil {
		gi = &greInfo{}
		name := filepath.Base(runMsg.Args[0])
		gi.Name = strings.TrimSuffix(name, filepath.Ext(name))
		gi.ID = genID(greIDWidth)
		gi.Args = runMsg.Args
		gi.Stat = greStatInit
		gi.RestartedNum = 0
		gi.AutoRestartBalance = runMsg.AutoRestartMax
		gi.RequestedBy = runMsg.RequestedBy
	}
	gc.greInfo = gi
	gc.statDir = filepath.Join(grg.statDir, gc.ID)
	gc.outputFile = filepath.Join(grg.workDir, "logs", gc.ID)
	of, err := os.Create(gc.outputFile)
	if err != nil {
		return nil, err
	}
	of.Close()

	if gc.Stat == greStatInit {
		if err := os.MkdirAll(gc.statDir, 0755); err != nil {
			return nil, err
		}
		if err := gc.runMsgToFile(); err != nil {
			return nil, err
		}
	}

	gsh, err := newShellWithCodeZip(runMsg.CodeZip)
	if err != nil {
		return nil, err
	}
	gc.gsh = gsh
	runMsg.CodeZip = nil // release the mem sooner
	gc.greInfoToFile()

	return gc, nil
}

func (grg *grg) runGRE(gc *greCtl) {
	for {
		gc.runGRE()
		if gc.AutoRestartBalance == 0 {
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

func (gc *greCtl) changeStat(newStat greStat) {
	atomic.StoreInt32(&gc.Stat, newStat)
}

func (gc *greCtl) changeStatIf(oldStat, newStat greStat) (changed bool) {
	return atomic.CompareAndSwapInt32(&gc.Stat, oldStat, newStat)
}

func (gc *greCtl) close() {
	os.Remove(gc.outputFile)
	os.RemoveAll(gc.statDir)
	gc.gsh.close()
}

func (gc *greCtl) runGRE() {
	gc.stderr = &strings.Builder{}
	gc.greErr = nil
	gc.GREErr = ""
	gc.StartTime = time.Now()
	gc.EndTime = time.Time{}
	gc.changeStat(greStatRunning)
	gc.greInfoToFile()

	defer func() {
		gc.EndTime = time.Now()
		stderrStr := gc.stderr.String()
		if len(stderrStr) != 0 {
			errstr := stderrStr
			index := strings.Index(errstr, "goroutine")
			if index != -1 {
				errstr = errstr[:index]
			}
			re := regexp.MustCompile(`os\.Exit\(.*\)`)
			if m := re.FindString(errstr); m != "" {
				if m == "os.Exit(0)" {
					//not an error
					stderrStr = ""
				} else {
					//user called os.Exit()
					stderrStr = fmt.Sprintln(m)
				}
			}
			if stderrStr != "" {
				gc.greErr = errors.New(stderrStr)
				gc.GREErr = stderrStr
			}
		}
		gc.stdin = nullIO{}
		gc.stdout = nil

		if gc.greErr == nil {
			gc.AutoRestartBalance = 0
		}
		if gc.AutoRestartBalance > 0 {
			gc.AutoRestartBalance--
		}
		gc.changeStatIf(greStatCancelling, greStatCancelled)
		gc.changeStatIf(greStatAborting, greStatAborted)
		gc.changeStatIf(greStatRunning, greStatExited)
		gc.greInfoToFile()
	}()

	log, err := os.OpenFile(gc.outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(gc.stderr, err)
		return
	}
	defer log.Close()

	if gc.stdout != nil {
		gc.stdout = multiWriter(gc.stdout, log)
	} else {
		gc.stdout = log
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc.cancel = cancel

	gsh := gc.gsh
	if err := gsh.init(interp.Options{
		Stdin:  gc.stdin,
		Stdout: gc.stdout,
		Stderr: gc.stderr,
		Args:   gc.Args,
	}); err != nil {
		fmt.Fprintln(gc.stderr, err)
	} else {
		if err := gsh.start(ctx); err != nil {
			fmt.Fprintln(gc.stderr, err)
			if p, ok := err.(interp.Panic); ok {
				fmt.Fprintln(gc.stderr, string(p.Stack))
			}
		}
	}
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
	RequestedBy string // by which provider ID
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
		gc.stdout = clientIO
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
type grgCmdJoblist struct {
	Tiny bool
}

func (msg grgCmdJoblist) Handle(stream as.ContextStream) (reply interface{}) {
	grg := stream.GetContext().(*grg)

	grgjl := &grgJoblist{
		Name:       grg.name,
		RtPriority: grg.rtPriority,
		Maxprocs:   grg.maxProcs,
	}

	grg.RLock()
	for _, gc := range grg.gres {
		func() {
			ji := &JobInfo{JobCmd: gc.runMsg.JobCmd}
			grgjl.Jobs = append(grgjl.Jobs, ji)
			if msg.Tiny {
				return
			}

			file, err := os.Open(filepath.Join(gc.statDir, "runMsg"))
			if err != nil {
				return
			}
			defer file.Close()
			runMsg := grgCmdRun{}
			if err := gob.NewDecoder(file).Decode(&runMsg); err != nil {
				return
			}
			ji.CodeZip = runMsg.CodeZip
		}()
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
			if gc.Stat == greStatRunning {
				if err := gc.gsh.stop(); err != nil {
					gc.gsh.abort(gc.cancel)
					gc.changeStat(greStatAborting)
				} else {
					gc.changeStat(greStatCancelling)
				}
				ids = append(ids, gc.ID)
			}
		case "rm":
			grg.lg.Infoln("checking rm GRE", gc.ID)
			if gc.Stat != greStatRunning {
				grg.lg.Infoln("rm GRE", gc.ID)
				grg.rmGRE(gc)
				ids = append(ids, gc.ID)
			}
		case "start":
			if greStatIsTerminated(gc.Stat) {
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
			if !greStatIsTerminated(gc.Stat) {
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
