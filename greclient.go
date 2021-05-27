package gshellos

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	gcStream = log.NewStream("greClient")
	gcLogger = gcStream.NewLogger("gre client", log.Lfatal)
)

type cmdTailf struct {
	Target string
}

func (c cmdTailf) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	io.Copy(os.Stdout, conn.GetNetConn())
	gcLogger.Traceln("cmdTailf: done")
	return io.EOF
}

type cmdPatternAction struct {
	GreName   string
	IDPattern []string
	Cmd       string
}

func (c cmdPatternAction) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		gcLogger.Errorln("recv error:", err)
		return err
	}
	ids := reply.([]*greVMIDs)
	var info string
	switch c.Cmd {
	case "kill":
		info = "killed"
	case "rm":
		info = "removed"
	case "restart":
		info = "restarted"
	}
	var sb strings.Builder
	for _, gvi := range ids {
		str := strings.Join(gvi.VMIDs, "\n")
		if len(str) != 0 {
			fmt.Fprintln(&sb, str)
		}
	}
	if sb.Len() > 0 {
		fmt.Print(sb.String())
		fmt.Println(info)
	}

	return io.EOF
}

type cmdQuery struct {
	GreName   string
	IDPattern []string
}

func (c cmdQuery) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		gcLogger.Errorln("recv error:", err)
		return err
	}
	gvis := reply.([]*greVMInfo)
	if len(c.IDPattern) != 0 { // info
		for _, gvi := range gvis {
			for _, vmi := range gvi.VMInfos {
				fmt.Println("ID        :", vmi.ID)
				fmt.Println("IN GRE    :", gvi.Name)
				fmt.Println("NAME      :", vmi.Name)
				fmt.Println("ARGS      :", vmi.Args[1:])
				fmt.Println("STATUS    :", vmi.Stat)
				if vmi.RestartedNum != 0 {
					fmt.Println("RESTARTED :", vmi.RestartedNum)
				}
				startTime := ""
				if !vmi.StartTime.IsZero() {
					startTime = fmt.Sprint(vmi.StartTime)
				}
				fmt.Println("START AT  :", startTime)
				endTime := ""
				if !vmi.EndTime.IsZero() {
					endTime = fmt.Sprint(vmi.EndTime)
				}
				fmt.Println("END AT    :", endTime)
				fmt.Printf("ERROR     : %v\n\n", vmi.VMErr)
			}
		}
	} else { // ps
		fmt.Println("VM ID         IN GRE        NAME          START AT             STATUS")
		trimName := func(name string) string {
			if len(name) > 12 {
				name = name[:9] + "..."
			}
			return name
		}
		for _, gvi := range gvis {
			for _, vmi := range gvi.VMInfos {

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

				fmt.Printf("%s  %-12s  %-12s  %s  %s\n", vmi.ID, trimName(gvi.Name), trimName(vmi.Name), created, stat)
			}
		}
	}

	return io.EOF
}

type greCmd struct {
	GreName string
	Cmd     string
	Paras   string
}

func (c greCmd) OnConnect(conn sm.Conn) error {
	if err := conn.Send(c); err != nil {
		gcLogger.Errorln("send cmd failed:", err)
		return err
	}
	reply, err := conn.Recv()
	if err != nil {
		gcLogger.Errorln("recv error:", err)
		return err
	}
	fmt.Println(reply)
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
	gcLogger.Traceln("connected to gre server")
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

	gcLogger.Traceln("sending run request")
	if err := conn.Send(c); err != nil {
		return err
	}

	if !c.Interactive {
		reply, err := conn.Recv()
		if err != nil {
			return err
		}
		fmt.Println(reply)
		return io.EOF
	}

	gcLogger.Traceln("enter interactive io")
	netconn := conn.GetNetConn()
	go io.Copy(netconn, os.Stdin)
	io.Copy(os.Stdout, netconn)
	gcLogger.Traceln("exit interactive io")
	return nil
}

func init() {
	gob.Register(&cmdRun{})
	gob.Register(cmdQuery{})
	gob.Register(cmdPatternAction{})
	gob.Register(cmdTailf{})
	gob.Register(greCmd{})
}
