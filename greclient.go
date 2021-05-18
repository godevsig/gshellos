package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	gcStream = log.NewStream("greClient")
	gcLogger = gcStream.NewLogger("gre client", log.Linfo)
)

type cmdPs struct {
	GreName string
}

func (c cmdPs) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		return err
	}
	gvis := reply.([]*greVMInfo)
	for _, gvi := range gvis {
		fmt.Println("gre:", gvi.Name)
		fmt.Println("VM ID         NAME          CREATED              STATUS")
		for _, vmi := range gvi.VMInfos {
			name := vmi.Name
			if len(name) > 12 {
				name = name[:12]
			}
			created := vmi.StartTime.Format("2006/01/02 15:04:05")
			stat := vmi.Stat
			switch stat {
			case "exited":
				ret := ":OK"
				if len(vmi.VMErr) != 0 {
					ret = ":ERR"
				}
				stat = stat + ret
				d := vmi.EndTime.Sub(vmi.StartTime)
				stat = fmt.Sprintf("%-10s %v", stat, d)
			case "running":
				d := time.Since(vmi.StartTime)
				stat = fmt.Sprintf("%-10s %v", stat, d)
			}

			fmt.Printf("%s  %-12s  %s  %s\n", vmi.ID, vmi.Name, created, stat)
		}
	}

	return io.EOF
}

type cmdRun struct {
	GreName     string
	File        string
	Args        []string
	Interactive bool
	AutoRemove  bool
	ByteCode    []byte
}

func (c *cmdRun) OnConnect(conn sm.Conn) error {
	gcLogger.Debugln("connected to gre server")
	sh := newShell()
	var b bytes.Buffer
	if filepath.Ext(c.File) == ".gsh" {
		bytecode, err := sh.compile(c.File)
		if err != nil {
			return err
		}
		err = bytecode.Encode(&b)
		if err != nil {
			return err
		}
	} else {
		f, err := os.Open(c.File)
		if err != nil {
			return err
		}
		defer f.Close()
		b.ReadFrom(f)
	}
	c.ByteCode = b.Bytes()

	gcLogger.Debugln("sending run request")
	if err := conn.Send(c); err != nil {
		return err
	}

	reply, err := conn.Recv()
	gcLogger.Debugln(reply, err)
	if err != nil {
		return err
	}

	if !c.Interactive {
		if !c.AutoRemove {
			fmt.Println(reply)
		}
		return io.EOF
	}
	return nil
}

type redirectMsg struct{}

func (redirectMsg) IsExclusive() {}
func (redirectMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	gcLogger.Debugln("enter interactive io")
	if err := conn.Send(redirectAckMsg{}); err != nil {
		fmt.Println(err)
		return nil, io.EOF
	}
	netconn := conn.GetNetConn()
	go io.Copy(netconn, os.Stdin)
	io.Copy(os.Stdout, netconn)
	gcLogger.Debugln("exit interactive io")
	return
}

func init() {
	gob.Register(&cmdRun{})
	gob.Register(redirectMsg{})
	gob.Register(cmdPs{})
}
