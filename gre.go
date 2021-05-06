package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/d5/tengo/v2"
	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	greStream = log.NewStream("gre")
	greLogger = greStream.NewLogger("gre", log.Linfo)
)

func init() {
	greStream.SetOutput("file:" + workDir + "gre.log")
}

type gre struct {
	name    string
	socket  string
	l       *sm.Listener
	errChan chan error
}

type session struct {
	vm *tengo.VM
}

func newgre(name string) (*gre, error) {
	greLogger.Infoln("starting gre", name)
	gre := &gre{name: name, errChan: make(chan error)}
	gre.socket = workDir + "gre-" + name + ".sock"
	l, err := sm.Listen("unix", gre.socket, sm.WithLogger(greLogger))
	if err != nil {
		greLogger.Errorln("gRE listen failed:", err)
		return nil, err
	}
	gre.l = l
	go func() {
		err := l.Run(gre)
		if err != nil {
			greLogger.Errorln("gRE run error:", err)
		}
		gre.errChan <- err
	}()

	return gre, nil
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

func (cmd *cmdRunMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	greLogger.Debugf("cmdRunMsg: file: %v, args: %v, interactive: %v\n", cmd.File, cmd.Args, cmd.Interactive)
	sh := newShell()
	bytecode := &tengo.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(cmd.ByteCode), sh.modules)
	if err != nil {
		greLogger.Errorln("cmdRunMsg: decode error:", err)
		return
	}

	vm := tengo.NewVM(bytecode, sh.globals, -1)
	vm.Args = cmd.Args
	if cmd.Interactive {
		session := conn.GetContext().(*session)
		session.vm = vm
		greLogger.Debugln("cmdRunMsg: sending redirectMsg")
		return redirectMsg{}, nil
	}

	greLogger.Debugln("cmdRunMsg: non-interactive")
	vmerr := vm.Run()
	if vmerr != nil {
		greLogger.Errorln("cmdRunMsg: vm run return error:", vmerr)
	}

	greLogger.Debugln("cmdRunMsg: closing connection")
	return nil, io.EOF
}

type redirectAckMsg struct{}

func (redirectAckMsg) IsExclusive() {}
func (redirectAckMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	greLogger.Debugln("redirectAckMsg: enter")
	session := conn.GetContext().(*session)
	vm := session.vm
	vm.In = conn.GetNetConn()
	vm.Out = conn.GetNetConn()

	vmerr := vm.Run()
	if vmerr != nil {
		fmt.Fprintln(conn.GetNetConn(), vmerr)
	}
	greLogger.Debugln("redirectAckMsg: closing")
	return nil, io.EOF
}

func init() {
	gob.Register(redirectAckMsg{})
	gob.Register(&cmdRunMsg{})
}
