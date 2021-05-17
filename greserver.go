package gshellos

import (
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	workDir  = "/var/tmp/gshell/"
	gsStream = log.NewStream("greServer")
	gsLogger = gsStream.NewLogger("server", log.Linfo)
	gserver  = &server{greConns: make(map[string]*greConn), errRecovers: make(chan errorRecover, 128)}
)

func init() {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		panic(err)
	}
	gsStream.SetOutput("file:" + workDir + "server.log")
}

type server struct {
	sync.RWMutex
	greConns    map[string]*greConn
	errRecovers chan errorRecover
}

type greConn struct {
	*gre
	sm.Conn // built-in conn to the gre
}

func (s *server) OnConnect(conn sm.Conn) error {
	gsLogger.Debugln("new connection", conn.GetNetConn().RemoteAddr().String())
	return nil
}

func setupgre(name string) (*greConn, error) {
	gre, err := newgre(name)
	if err != nil {
		gsLogger.Errorf("newgre %s failed: %v", name, err)
		return nil, err
	}
	fakeRecover := func() bool {
		return true
	}
	go func() {
		var err error
		defer func() {
			gsLogger.Errorf("%s gre terminated unexpectedly", name)
			gserver.Lock()
			delete(gserver.greConns, name)
			gserver.Unlock()
			gre.clean()
			if perr := recover(); perr != nil {
				gsLogger.Errorf("%s gre runtime panic: %v", name, perr)
			}
			// not attempt to restart the gre
			gserver.errRecovers <- customErrorRecover{err, name + " gre runtime error", fakeRecover}
		}()

		err = gre.run() // gres are not supposed to exit
	}()

	greConnLogger := gsStream.GetLogger(name + " conn")
	if greConnLogger == nil {
		greConnLogger = gsStream.NewLogger(name+" conn", log.Linfo)
	}

	grecChan := make(chan *greConn)
	onConnect := func(conn sm.Conn) error {
		grec := &greConn{gre, conn}
		gserver.Lock()
		gserver.greConns[name] = grec
		gserver.Unlock()
		grecChan <- grec
		return nil
	}

	go func() {
		var err error
		defer func() {
			gserver.Lock()
			delete(gserver.greConns, name)
			gserver.Unlock()
			gsLogger.Infof("closing gre %s due to connection lost", name)
			gre.close() // shut down the gre
			if perr := recover(); perr != nil {
				gsLogger.Errorf("builtin connection to %s runtime panic: %v", name, perr)
			}
			gserver.errRecovers <- customErrorRecover{err, name + " builtin connection error", fakeRecover}
		}()

		if _, err := os.Stat(gre.socket); os.IsNotExist(err) {
			time.Sleep(time.Second)
		}
		// built-in conn to gre, not supposed to exit
		err = sm.DialRun(sm.OnConnectFunc(onConnect), "unix", gre.socket, sm.WithLogger(greConnLogger))
	}()

	grec := <-grecChan
	gsLogger.Infof("%s gre created\n", name)
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

	cleanOldgserver := func() {
		defer func() {
			sockets, _ := filepath.Glob(workDir + "*.sock")
			for _, s := range sockets {
				os.Remove(s)
			}
		}()

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
	cleanOldgserver()
	pid := os.Getpid() // new pid
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return errorHere(err)
	}
	defer func() {
		if getPidFromFile() == pid {
			os.Remove(pidFile)
		}
	}()

	if len(port) != 0 {
		go func() {
			gsLogger.Infoln("listening on tcp port", port)
			err := sm.ListenRun(gserver, "tcp", ":"+port, sm.WithLogger(gsLogger))
			gserver.errRecovers <- unrecoverableError{err}
		}()
	}
	go func() {
		gsLogger.Infoln("listening on local unix")
		err := sm.ListenRun(gserver, "unix", workDir+"gshelld.sock", sm.WithLogger(gsLogger))
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

type reqRunMsg struct {
	GreName     string
	File        string
	Args        []string
	Interactive bool
	AutoRemove  bool
	ByteCode    []byte
}

func (*reqRunMsg) IsExclusive() {}
func (req *reqRunMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	gsLogger.Debugf("reqRunMsg: file: %v, args: %v, interactive: %v", req.File, req.Args, req.Interactive)

	gserver.RLock()
	grec, ok := gserver.greConns[req.GreName]
	gserver.RUnlock()
	if !ok {
		tgrec, err := setupgre(req.GreName)
		//err = errors.New("test") // for test
		if err != nil {
			gsLogger.Errorf("reqRunMsg: setup gre %s failed: %v", req.GreName, err)
			return nil, errors.New("create gre failed")
		}
		grec = tgrec
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

	err := sm.DialRun(sm.OnConnectFunc(onConnect), "unix", grec.socket, sm.RawMode(), sm.ErrorAsEOF())
	if err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func init() {
	gob.Register(&reqRunMsg{})
}
