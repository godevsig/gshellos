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

type cmdRunMsg struct {
	File        string
	Args        []string
	Interactive bool
	ByteCode    []byte
}

func genVMID() string {
	b := make([]byte, 6)
	rand.Read(b)
	id := hex.EncodeToString(b)
	return id
}

func (gre *gre) addVM(vc *vmCtl) {
	gre.Lock()
	gre.vms[vc.id] = vc
	gre.Unlock()
}

func (gre *gre) rmVM(vc *vmCtl) {
	gre.Lock()
	delete(gre.vms, vc.id)
	gre.Unlock()
}

type vmCtl struct {
	sync.Mutex
	*tengo.VM
	vmErr     error // returned error when VM exits
	name      string
	id        string
	runMsg    *cmdRunMsg
	stat      string // starting running exited
	startTime time.Time
	endTime   time.Time
}

type null struct{}

func (null) Close() error                  { return nil }
func (null) Write(buf []byte) (int, error) { return len(buf), nil }
func (null) Read(buf []byte) (int, error)  { return 0, io.EOF }

func (cmd *cmdRunMsg) Handle(conn sm.Conn) (reply interface{}, retErr error) {
	greLogger.Debugf("cmdRunMsg: file: %v, args: %v, interactive: %v\n", cmd.File, cmd.Args, cmd.Interactive)
	sh := newShell()
	bytecode := &tengo.Bytecode{}
	err := bytecode.Decode(bytes.NewReader(cmd.ByteCode), sh.modules)
	err = errors.New("test")
	if err != nil {
		greLogger.Errorf("cmdRunMsg: decode %s error: %v", cmd.File, err)
		return nil, errors.New("bad bytecode")
	}

	name := filepath.Base(cmd.File)
	vm := tengo.NewVM(bytecode, sh.globals, -1)
	vm.Args = cmd.Args
	vc := &vmCtl{
		VM:     vm,
		name:   strings.TrimSuffix(name, filepath.Ext(name)),
		id:     genVMID(),
		runMsg: cmd,
		stat:   "starting",
	}

	ggre.addVM(vc)

	if cmd.Interactive {
		session := conn.GetContext().(*session)
		session.vc = vc
		greLogger.Traceln("cmdRunMsg: sending redirectMsg")
		return redirectMsg{}, nil
	}

	go func() {
		output, err := os.Create(logDir + vc.id)
		if err != nil {
			greLogger.Errorln("create output file error:", err)
			return
		}
		defer output.Close()
		vm.In = null{}
		vm.Out = output
		greLogger.Traceln("cmdRunMsg: non-interactive")
		vc.startTime = time.Now()
		vc.stat = "running"
		err = vm.Run()
		vc.endTime = time.Now()
		if err != nil {
			fmt.Fprintln(output, err)
		}
		vc.vmErr = err
		vc.stat = "exited"
	}()

	return vc.id, nil
}

type redirectAckMsg struct{}

func (redirectAckMsg) IsExclusive() {}
func (redirectAckMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	greLogger.Traceln("redirectAckMsg: enter")
	session := conn.GetContext().(*session)
	vm := session.vc.VM
	vm.In = conn.GetNetConn()
	vm.Out = conn.GetNetConn()

	vmerr := vm.Run()
	if vmerr != nil {
		fmt.Fprintln(conn.GetNetConn(), vmerr)
	}
	greLogger.Traceln("redirectAckMsg: closing")
	return nil, io.EOF
}

func init() {
	gob.Register(redirectAckMsg{})
	gob.Register(&cmdRunMsg{})
}
