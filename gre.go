package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/d5/tengo/v2"
	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	greLogger *log.Logger
	greStream = log.NewStream("gre")
)

func init() {
	// same file shared by multiple gshell processes in append mode.
	greStream.SetOutput("file:" + workDir + "gre.log")
}

type gre struct {
	name   string
	socket string
	l      *sm.Listener
}

type session struct {
	vm *tengo.VM
}

func newgre(name string) (*gre, error) {
	if greLogger = greStream.GetLogger(name); greLogger == nil {
		greLogger = greStream.NewLogger(name, log.Linfo)
	}

	gre := &gre{
		name:   name,
		socket: workDir + "gre-" + name + ".sock",
	}
	l, err := sm.Listen("unix", gre.socket, sm.WithLogger(greLogger))
	if err != nil {
		return nil, err
	}
	gre.l = l
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
		greLogger.Traceln("cmdRunMsg: sending redirectMsg")
		return redirectMsg{}, nil
	}

	greLogger.Traceln("cmdRunMsg: non-interactive")
	vmerr := vm.Run()
	if vmerr != nil {
		greLogger.Errorln("cmdRunMsg: vm run return error:", vmerr)
	}

	return nil, nil
}

type redirectAckMsg struct{}

func (redirectAckMsg) IsExclusive() {}
func (redirectAckMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	greLogger.Traceln("redirectAckMsg: enter")
	session := conn.GetContext().(*session)
	vm := session.vm
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
