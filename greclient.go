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

type cmdKill struct {
	GreName  string
	IDPatten []string
}

func (c cmdKill) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		gcLogger.Errorln("recv failed:", err)
		return err
	}
	killed := reply.([]*greVMIDs)
	for _, gvi := range killed {
		fmt.Println(gvi)
	}
	return io.EOF
}

type cmdQuery struct {
	GreName  string
	IDPatten []string
}

func (c cmdQuery) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		gcLogger.Errorln("recv failed:", err)
		return err
	}
	gvis := reply.([]*greVMInfo)
	if len(c.IDPatten) != 0 { // info
		for _, gvi := range gvis {
			fmt.Println("===")
			fmt.Println("gre:", gvi.Name)
			for _, vmi := range gvi.VMInfos {
				fmt.Println("ID      :", vmi.ID)
				fmt.Println("NAME    :", vmi.Name)
				fmt.Println("ARGS    :", vmi.Args[1:])
				fmt.Println("STATUS  :", vmi.Stat)
				startTime := ""
				if !vmi.StartTime.IsZero() {
					startTime = fmt.Sprint(vmi.StartTime)
				}
				fmt.Println("START AT:", startTime)
				endTime := ""
				if !vmi.EndTime.IsZero() {
					endTime = fmt.Sprint(vmi.EndTime)
				}
				fmt.Println("END AT  :", endTime)
				fmt.Printf("ERROR   : %v\n\n", vmi.VMErr)
			}
		}
	} else { // ps
		for _, gvi := range gvis {
			fmt.Println("===")
			fmt.Println("gre:", gvi.Name)
			fmt.Println("VM ID         NAME          START AT             STATUS")
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
	gob.Register(cmdQuery{})
	gob.Register(cmdKill{})
}
