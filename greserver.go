package gshellos

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	workDir  = "/var/tmp/gshell/"
	gsStream = log.NewStream("greServer")
	gsLogger = gsStream.NewLogger("server", log.Linfo)
	gserver  = &server{greCtls: make(map[string]*greCtl), errRecovers: make(chan errorRecover, 128)}
)

func init() {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		panic(err)
	}
	gsStream.SetOutput("file:" + workDir + "server.log")
}

type server struct {
	sync.RWMutex
	greCtls     map[string]*greCtl
	errRecovers chan errorRecover
}

type greCtl struct {
	name           string
	addr           string
	p              *os.Process
	suspended      int32
	savedGreVMInfo *greVMInfo
	sm.Conn        // built-in conn to the gre
}

func (s *server) addGreCtl(name string, grec *greCtl) {
	s.Lock()
	s.greCtls[name] = grec
	s.Unlock()
}

func (s *server) rmGreCtl(name string) {
	s.Lock()
	delete(s.greCtls, name)
	s.Unlock()
}

func (s *server) getGreCtl(name string) *greCtl {
	s.RLock()
	grec := s.greCtls[name]
	s.RUnlock()
	return grec
}

func (s *server) OnConnect(conn sm.Conn) error {
	gsLogger.Debugln("new connection", conn.GetNetConn().RemoteAddr().String())
	return nil
}

func setupgre(name string) (*greCtl, error) {
	greConnLogger := gsStream.GetLogger(name + " conn")
	if greConnLogger == nil {
		greConnLogger = gsStream.NewLogger(name+" conn", log.Linfo)
	}
	addr := workDir + "gre-" + name + ".sock"
	dlr, err := sm.Dial("unix", addr, sm.WithLogger(greConnLogger))
	if err != nil { // gre not running yet
		if name == "master" {
			err = fmt.Errorf("dial master gre failed: %w", err)
			gserver.errRecovers <- unrecoverableError{err}
			return nil, err
		}
		args := savedOptions + "-e " + name + " gre " + "__start"
		cmd := exec.Command(os.Args[0], strings.Split(args, " ")...)
		gsLogger.Debugln("starting:", cmd.String())
		err = cmd.Start()
		if err != nil {
			err = fmt.Errorf("start %s gre failed: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
			return nil, err
		}

		dlr, err = sm.Dial("unix", addr, sm.WithLogger(greConnLogger), sm.AutoWait(10))
		if err != nil {
			err = fmt.Errorf("dial %s gre failed: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
			return nil, err
		}
	}

	grecChan := make(chan *greCtl)
	onConnect := func(conn sm.Conn) error {
		grec := &greCtl{name: name, addr: addr, Conn: conn}
		if err := conn.Send(getProcess{}); err != nil {
			err = fmt.Errorf("get %s gre pid send failed: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
			return io.EOF
		}
		reply, err := conn.Recv()
		if err != nil {
			err = fmt.Errorf("get %s gre pid failed: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
			return io.EOF
		}
		pid := reply.(int)
		grec.p, err = os.FindProcess(pid)
		if err != nil {
			err = fmt.Errorf("find %s gre pid failed: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
			return io.EOF
		}
		gsLogger.Infof("%s gre is up", name)
		gserver.addGreCtl(name, grec)
		grecChan <- grec
		return nil
	}

	go func() {
		defer func() {
			gserver.rmGreCtl(name)
			gsLogger.Infof("%s gre is down", name)
		}()
		if err := dlr.Run(sm.OnConnectFunc(onConnect)); err != nil {
			err = fmt.Errorf("connection to %s gre encountered error: %w", name, err)
			gserver.errRecovers <- unrecoverableError{err}
		}
	}()

	grec := <-grecChan
	gsLogger.Infof("%s gre added\n", name)
	return grec, nil
}

func runServer(version, port string) error {
	gsLogger.Infof("gshell server version: %s", version)
	pidFile := workDir + "gserver.pid"

	getPidFromFile := func() int {
		data, err := ioutil.ReadFile(pidFile)
		if err != nil {
			return 0
		}
		pid, err := strconv.Atoi(string(data))
		if err != nil {
			return 0
		}
		return pid
	}

	killOldgserver := func() {
		pid := getPidFromFile()
		if pid == 0 {
			return
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		gsLogger.Infoln("shutting down old gre server", pid)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			gsLogger.Errorf("kill pid %d failed: %s", pid, err)
		}
	}
	killOldgserver()
	pid := os.Getpid() // new pid
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return errorHere(err)
	}
	defer func() {
		if getPidFromFile() == pid {
			os.Remove(pidFile)
		}
	}()

	master, err := newgre("master")
	if err != nil {
		gsLogger.Errorln("creating default master gre failed")
		return err
	}
	go func() {
		gsLogger.Infoln("running default master gre")
		err := master.run()
		gserver.errRecovers <- unrecoverableError{err}
	}()

	sockets, _ := filepath.Glob(workDir + "gre-*.sock")
	for _, s := range sockets {
		s = strings.TrimPrefix(s, workDir+"gre-")
		s = strings.TrimSuffix(s, ".sock")
		gsLogger.Infof("connect to %s gre", s)
		if _, err := setupgre(s); err != nil {
			gsLogger.Warnln(s + " gre setup failed")
		}
	}

	if len(port) != 0 {
		go func() {
			gsLogger.Infoln("listening on tcp port", port)
			err := sm.ListenRun(gserver, "tcp", ":"+port, sm.WithLogger(gsLogger))
			gserver.errRecovers <- unrecoverableError{err}
		}()
	}

	addr := workDir + "gshelld.sock"
	defer os.Remove(addr)
	go func() {
		gsLogger.Infoln("listening on local unix")
		err := sm.ListenRun(gserver, "unix", addr, sm.WithLogger(gsLogger))
		gserver.errRecovers <- unrecoverableError{err}
	}()

	for e := range gserver.errRecovers {
		if e.Recover() {
			gsLogger.Infoln("recovered:", e.String(), ":", e.Error())
		} else {
			gsLogger.Errorln("not recovered:", e.String(), ":", e.Error())
			return e.Error()
		}
	}
	return nil
}

// return specified gre or all gres
func greCtlsByName(greName string) (grecs []*greCtl) {
	if len(greName) != 0 {
		grec := gserver.getGreCtl(greName)
		if grec != nil {
			grecs = append(grecs, grec)
		}
	} else {
		names := make([]string, 0, len(gserver.greCtls))
		gserver.RLock()
		for name := range gserver.greCtls {
			names = append(names, name)
		}
		gserver.RUnlock()
		sort.Strings(names)
		for _, name := range names {
			grec := gserver.getGreCtl(name)
			if grec != nil {
				grecs = append(grecs, grec)
			}
		}
	}
	return
}

type reqTailfMsg = cmdTailf

type endlessReader struct {
	r io.Reader
}

func (er endlessReader) Read(p []byte) (n int, err error) {
	for i := 0; i < 30; i++ {
		n, err = er.r.Read(p)
		if err != io.EOF {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	p[n] = 0 // fake read
	return n + 1, nil
}

func (req reqTailfMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	retErr = io.EOF
	var file string
	switch req.Target {
	case "server":
		file = workDir + "server.log"
	case "gre":
		file = workDir + "gre.log"
	default:
		file = workDir + "logs/" + req.Target
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(conn.GetNetConn(), req.Target+" not found")
		return
	}
	io.Copy(conn.GetNetConn(), endlessReader{f})
	gsLogger.Traceln("reqTailfMsg: done")
	return
}

type reqPatternActionMsg = cmdPatternAction

type greVMIDs struct {
	Name  string
	VMIDs []string
}

func (req reqPatternActionMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	grecs := greCtlsByName(req.GreName)
	if len(grecs) == 0 {
		return nil, errors.New(req.GreName + " gre not found")
	}

	pattenAction := func(grec *greCtl) (*greVMIDs, error) {
		if err := grec.Send(cmdPatternActionMsg{req.IDPattern, req.Cmd}); err != nil {
			gsLogger.Errorf("pattenAction: send cmd to gre %s failed: %v", grec.name, err)
			return nil, err
		}
		ids, err := grec.Recv()
		if err != nil {
			gsLogger.Errorf("pattenAction: recv from gre %s failed: %v", grec.name, err)
			return nil, err
		}
		return &greVMIDs{grec.name, ids.([]string)}, nil
	}

	var ids []*greVMIDs
	for _, grec := range grecs {
		if atomic.LoadInt32(&grec.suspended) == 1 {
			continue
		}
		gvi, err := pattenAction(grec)
		if err != nil {
			gsLogger.Errorln(errorHere(err))
			continue
		}
		ids = append(ids, gvi)
	}
	return ids, nil
}

type reqQueryMsg = cmdQuery

func getGreVMInfo(grec *greCtl, IDPattern []string) (*greVMInfo, error) {
	if err := grec.Send(cmdQueryMsg{IDPattern}); err != nil {
		gsLogger.Errorf("getGreVMInfo: send cmd to gre %s failed: %v", grec.name, err)
		return nil, err
	}
	gvi, err := grec.Recv()
	if err != nil {
		gsLogger.Errorf("getGreVMInfo: recv from gre %s failed: %v", grec.name, err)
		return nil, err
	}
	return gvi.(*greVMInfo), nil
}

func (req reqQueryMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	var gvis []*greVMInfo
	grecs := greCtlsByName(req.GreName)
	if len(grecs) == 0 {
		if len(req.GreName) == 0 {
			return gvis, nil
		}
		return nil, errors.New("gre not found")
	}

	for _, grec := range grecs {
		var gvi *greVMInfo
		var err error
		if atomic.LoadInt32(&grec.suspended) == 1 {
			gvi = grec.savedGreVMInfo
		} else {
			gvi, err = getGreVMInfo(grec, req.IDPattern)
		}
		if err != nil {
			gsLogger.Errorln(errorHere(err))
			continue
		}
		gvis = append(gvis, gvi)
	}
	return gvis, nil
}

type greCmdReq = greCmd

func (req greCmdReq) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	grec := gserver.getGreCtl(req.GreName)
	if grec == nil {
		return nil, errors.New(req.GreName + " gre not found")
	}

	switch req.Cmd {
	case "stop":
		if req.Paras != "-f" {
			if atomic.LoadInt32(&grec.suspended) == 1 {
				return nil, errors.New(req.GreName + " gre suspended")
			}
			gvmi, err := getGreVMInfo(grec, nil)
			if err != nil {
				return nil, err
			}
			for _, vmi := range gvmi.VMInfos {
				if vmi.Stat != "exited" {
					return nil, errors.New("not all VMs are exited")
				}
			}
		}
		if err := grec.p.Signal(syscall.SIGTERM); err != nil {
			return nil, err
		}
	case "save":
		return "to be added", nil
	case "load":
		return "to be added", nil
	case "suspend":
		if atomic.LoadInt32(&grec.suspended) == 0 {
			gvi, err := getGreVMInfo(grec, nil)
			if err != nil {
				return nil, err
			}
			for _, vmi := range gvi.VMInfos {
				if vmi.Stat != "exited" {
					vmi.Stat = "suspended"
				}
			}
			grec.savedGreVMInfo = gvi
			atomic.StoreInt32(&grec.suspended, 1)
			if err := grec.p.Signal(syscall.SIGSTOP); err != nil {
				return nil, err
			}
		}
	case "resume":
		if atomic.LoadInt32(&grec.suspended) == 1 {
			if err := grec.p.Signal(syscall.SIGCONT); err != nil {
				return nil, err
			}
			grec.savedGreVMInfo = nil
			atomic.StoreInt32(&grec.suspended, 0)
		}
	case "priority":
		return "to be added", nil
	default:
		return "unknown command", nil
	}
	return "ok", nil
}

type reqRunMsg = cmdRun

func (*reqRunMsg) IsExclusive() {}
func (req *reqRunMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	gsLogger.Debugf("reqRunMsg: file: %v, args: %v, interactive: %v", req.File, req.Args, req.Interactive)

	grec := gserver.getGreCtl(req.GreName)
	if grec == nil {
		tgrec, err := setupgre(req.GreName)
		//err = errors.New("test") // for test
		if err != nil {
			gsLogger.Errorf("reqRunMsg: setup gre %s failed: %v", req.GreName, err)
			return nil, errors.New("create " + req.GreName + " gre failed")
		}
		grec = tgrec
	}
	if atomic.LoadInt32(&grec.suspended) == 1 {
		return nil, errors.New(req.GreName + " gre suspended")
	}

	cmd := &cmdRunMsg{
		File:        req.File,
		Args:        req.Args,
		Interactive: req.Interactive,
		AutoRemove:  req.AutoRemove,
		ByteCode:    req.ByteCode,
	}

	if !req.Interactive {
		if err := grec.Send(cmd); err != nil {
			gsLogger.Errorf("reqRunMsg: send cmd to gre %s failed: %v", req.GreName, err)
			return nil, errors.New("send cmd to gre failed")
		}
		return grec.Recv()
	}

	onConnect := func(c sm.Conn) error {
		if err := c.Send(cmd); err != nil {
			gsLogger.Errorf("reqRunMsg: send cmd to gre %s failed: %v", req.GreName, err)
			conn.Send(errors.New("send cmd to gre failed"))
			return io.EOF
		}
		msg, err := c.Recv()
		if err != nil {
			gsLogger.Errorf("reqRunMsg: recv from gre %s failed: %v", req.GreName, err)
			conn.Send(err)
			return io.EOF
		}
		conn.Send(msg)
		gsLogger.Traceln("reqRunMsg: forwarding io")
		done := make(chan struct{}, 2)
		copy := func(dst io.Writer, src io.Reader) {
			io.Copy(dst, src)
			done <- struct{}{}
		}
		go copy(c.GetNetConn(), conn.GetNetConn())
		go copy(conn.GetNetConn(), c.GetNetConn())

		<-done
		gsLogger.Traceln("reqRunMsg: closing connection")
		return io.EOF
	}

	err := sm.DialRun(sm.OnConnectFunc(onConnect), "unix", grec.addr, sm.RawMode(), sm.ErrorAsEOF())
	if err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func init() {
	gob.Register([]*greVMInfo{})
	gob.Register([]*greVMIDs{})
}
