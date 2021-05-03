package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/d5/tengo/v2"
	sm "github.com/godevsig/gshellos/scalamsg"
)

type server struct {
}

type session struct {
	vm *tengo.VM
}

func (s *server) OnConnect(conn sm.Conn) error {
	ctx := &session{}
	conn.SetContext(ctx)
	return nil
}

func runServer(port string) error {
	s := &server{}
	errChan := make(chan error)
	if len(port) != 0 {
		go func() {
			err := sm.ListenRun("tcp", fmt.Sprintf(":%s", port), s)
			errChan <- err
		}()
	}
	go func() {
		err := sm.ListenRun("unix", "/var/tmp/gshelld.sock", s)
		errChan <- err
	}()

	return <-errChan
}

type reqRunMsg struct {
	GreName     string
	File        string
	Args        []string
	Interactive bool
	ByteCode    []byte
}

func (req *reqRunMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	sh := newShell()
	bytecode := &tengo.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(req.ByteCode), sh.modules)
	if err != nil {
		return
	}

	vm := tengo.NewVM(bytecode, sh.globals, -1)
	if req.Interactive {
		session := conn.GetContext().(*session)
		session.vm = vm
		return redirectMsg{}, nil
	}

	vmerr := vm.Run()
	if vmerr != nil {
		fmt.Println(vmerr)
	}
	return
}

type redirectAckMsg struct{}

func (redirectAckMsg) IsExclusive() {}
func (redirectAckMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	session := conn.GetContext().(*session)
	vm := session.vm
	vm.In = conn.GetNetConn()
	vm.Out = conn.GetNetConn()

	vmerr := vm.Run()
	if vmerr != nil {
		fmt.Fprintln(conn.GetNetConn(), vmerr)
	}
	return nil, io.EOF
}

func init() {
	gob.Register(&reqRunMsg{})
	gob.Register(redirectAckMsg{})
}
