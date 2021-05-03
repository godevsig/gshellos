package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"

	sm "github.com/godevsig/gshellos/scalamsg"
)

type cmdRun struct {
	greName     string
	file        string
	args        []string
	interactive bool
}

func (c *cmdRun) OnConnect(conn sm.Conn) error {
	sh := newShell()
	var b bytes.Buffer
	if filepath.Ext(c.file) == ".gsh" {
		bytecode, err := sh.compile(c.file)
		if err != nil {
			return err
		}
		err = bytecode.Encode(&b)
		if err != nil {
			return err
		}
	} else {
		f, err := os.Open(c.file)
		if err != nil {
			return err
		}
		defer f.Close()
		b.ReadFrom(f)
	}

	msg := &reqRunMsg{
		GreName:     c.greName,
		File:        c.file,
		Args:        c.args,
		Interactive: c.interactive,
		ByteCode:    b.Bytes(),
	}

	if err := conn.Send(msg); err != nil {
		fmt.Println(err)
		return io.EOF
	}
	if !c.interactive {
		return io.EOF
	}
	return nil
}

type redirectMsg struct{}

func (redirectMsg) IsExclusive() {}
func (redirectMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	if err := conn.Send(redirectAckMsg{}); err != nil {
		fmt.Println(err)
		return nil, io.EOF
	}
	netconn := conn.GetNetConn()
	go io.Copy(netconn, os.Stdin)
	io.Copy(os.Stdout, netconn)
	return
}

func init() {
	gob.Register(redirectMsg{})
}
