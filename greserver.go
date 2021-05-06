package gshellos

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var (
	workDir  = "/var/tmp/gshell/"
	gsStream = log.NewStream("greServer")
	gsLogger = gsStream.NewLogger("gre server", log.Linfo)
	gserver  = &server{gres: make(map[string]*gre)}
)

func init() {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		panic(err)
	}
	gsStream.SetOutput("file:" + workDir + "greserver.log")
}

type server struct {
	gres map[string]*gre
}

func (s *server) OnConnect(conn sm.Conn) error {
	gsLogger.Debugln("new connection", conn.GetNetConn().RemoteAddr().String())
	return nil
}

func runServer(port string) error {
	errChan := make(chan error)
	if len(port) != 0 {
		go func() {
			gsLogger.Infoln("starting listening on tcp port", port)
			err := sm.ListenRun(gserver, "tcp", ":"+port, sm.WithLogger(gsLogger))
			errChan <- err
		}()
	}
	go func() {
		gsLogger.Infoln("starting listening on local unix")
		err := sm.ListenRun(gserver, "unix", workDir+"gshelld.sock", sm.WithLogger(gsLogger))
		errChan <- err
	}()

	master, err := newgre("master")
	if err != nil {
		return err
	}
	gserver.gres["master"] = master
	gsLogger.Infoln("gre master created")

	greErr := <-master.errChan
	serverErr := <-errChan

	if greErr == nil && serverErr == nil {
		return nil
	}
	return fmt.Errorf("server error: %v\ngre master error: %v", serverErr, greErr)
}

type reqRunMsg struct {
	GreName     string
	File        string
	Args        []string
	Interactive bool
	ByteCode    []byte
}

func (*reqRunMsg) IsExclusive() {}
func (req *reqRunMsg) Handle(conn sm.Conn) (reply interface{}, err error) {
	gsLogger.Debugf("reqRunMsg: file: %v, args: %v, interactive: %v\n", req.File, req.Args, req.Interactive)
	if req.GreName != "master" {
		panic("ToDo: add named gre")
	}

	c, err := net.Dial("unix", gserver.gres["master"].socket)
	if err != nil {
		gsLogger.Errorln("reqRunMsg: dial failed:", err)
		return nil, err
	}
	defer c.Close()

	var cmd interface{}
	cmd = &cmdRunMsg{
		File:        req.File,
		Args:        req.Args,
		Interactive: req.Interactive,
		ByteCode:    req.ByteCode,
	}
	enc := gob.NewEncoder(c)
	if err := enc.Encode(&cmd); err != nil {
		gsLogger.Errorln("reqRunMsg: cmd not sent:", err)
		return nil, io.EOF
	}

	if !req.Interactive {
		return nil, io.EOF
	}

	gsLogger.Debugln("reqRunMsg: forwarding io")
	done := make(chan struct{}, 2)
	copy := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go copy(c, conn.GetNetConn())
	go copy(conn.GetNetConn(), c)

	<-done
	gsLogger.Debugln("reqRunMsg: closing connection")
	return nil, io.EOF
}

func init() {
	gob.Register(&reqRunMsg{})
}
