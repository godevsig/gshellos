package gshellos

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/gshellos/log"
)

type daemon struct {
	lg *log.Logger
}

func (gd *daemon) onNewStream(ctx as.Context) {
	ctx.SetContext(gd)
}

func (gd *daemon) setupgre(name string) as.Connection {
	opts := []as.Option{
		as.WithLogger(gd.lg),
		as.WithScope(as.ScopeOS),
	}
	c := as.NewClient(opts...).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "gre-"+name)
	if conn != nil {
		return conn
	}

	args := "-wd " + workDir + " -loglevel " + loglevel + " __start " + "-e " + name
	if os.Args[0] == "gshell.tester" {
		args = "-test.run ^TestRunMain$ -test.coverprofile=.test/l2_gre" + name + ".cov -- " + args
	}
	cmd := exec.Command(os.Args[0], strings.Split(args, " ")...)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	gd.lg.Debugln("starting gre:", cmd.String())

	if err := cmd.Start(); err != nil {
		gd.lg.Errorf("start cmd %s failed: %w", cmd.String(), err)
		return nil
	}
	go func() {
		err := cmd.Wait()
		if err != nil {
			gd.lg.Errorf("cmd: %s exited with error: %v, output: %v", cmd.String(), err, buf.String())
		} else {
			gd.lg.Infof("cmd: %s exited, output: %v", cmd.String(), buf.String())
		}
	}()

	c.SetDiscoverTimeout(3)
	return <-c.Discover(godevsigPublisher, "gre-"+name)
}

type cmdRun struct {
	greCmdRun
	GreName string
}

func (msg *cmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdRun: file %v, args %v, interactive %v", msg.File, msg.Args, msg.Interactive)

	conn := gd.setupgre(msg.GreName)
	if conn == nil {
		return ErrBrokenGre
	}
	defer conn.Close()

	if err := conn.Send(&msg.greCmdRun); err != nil {
		return err
	}

	if !msg.Interactive {
		var vmid string
		if err := conn.Recv(&vmid); err != nil {
			return err
		}
		return vmid
	}
	gd.lg.Debugln("enter interactive io")
	go io.Copy(conn, stream)
	io.Copy(stream, conn)
	gd.lg.Debugln("exit interactive io")

	return io.EOF
}

type cmdQuery struct {
	GreName   string
	IDPattern []string
}

func (msg *cmdQuery) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdQuery: %v", msg)

	var gvis []*greVMInfo
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	connChan := c.Discover(godevsigPublisher, "gre-"+msg.GreName)
	for conn := range connChan {
		var gvi *greVMInfo
		if err := conn.SendRecv(&greCmdQuery{msg.IDPattern}, &gvi); err != nil {
			gd.lg.Warnf("cmdQuery: send recv error: %v", err)
		}
		if gvi != nil {
			gvis = append(gvis, gvi)
		}
		conn.Close()
	}
	return gvis
}

type cmdPatternAction struct {
	GreName   string
	IDPattern []string
	Cmd       string
}

type greVMIDs struct {
	//Name  string
	VMIDs []string
}

func (msg *cmdPatternAction) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdPatternAction: %v", msg)

	var gvmids []*greVMIDs
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	connChan := c.Discover(godevsigPublisher, "gre-"+msg.GreName)
	for conn := range connChan {
		var vmids []string
		if err := conn.SendRecv(&greCmdPatternAction{msg.IDPattern, msg.Cmd}, &vmids); err != nil {
			gd.lg.Warnf("cmdPatternAction: send recv error: %v", err)
		}
		if vmids != nil {
			gvmids = append(gvmids, &greVMIDs{VMIDs: vmids})
		}
		conn.Close()
	}
	return gvmids
}

type cmdTailf struct {
	Target string
}

func (msg *cmdTailf) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	reply = io.EOF
	var file string
	switch msg.Target {
	case "daemon":
		file = workDir + "/logs/daemon.log"
	case "gre":
		file = workDir + "/logs/gre.log"
	default:
		file = workDir + "/logs/" + msg.Target
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(stream, msg.Target+" not found")
		return
	}
	io.Copy(stream, endlessReader{f})
	gd.lg.Debugln("cmdTailf: done")
	return
}

type cmdInfo struct{}

func (msg cmdInfo) Handle(stream as.ContextStream) (reply interface{}) {
	var b strings.Builder
	fmt.Fprintf(&b, "Version: %s\n", version)
	fmt.Fprintf(&b, "Build tags: %s\n", buildTags)

	return b.String()
}

var daemonKnownMsgs = []as.KnownMessage{
	(*cmdRun)(nil),
	(*cmdQuery)(nil),
	(*cmdPatternAction)(nil),
	(*cmdTailf)(nil),
	cmdInfo{},
}

type codeRepoSvc struct {
	repoInfo []string
}

type codeRepoAddr struct{}

func (msg codeRepoAddr) Handle(stream as.ContextStream) (reply interface{}) {
	crs := stream.GetContext().(*codeRepoSvc)
	return strings.Join(crs.repoInfo[:3], "/") + " " + crs.repoInfo[3]
}

type getFileContent struct {
	File string
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("file not found: %d error", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (msg getFileContent) Handle(stream as.ContextStream) (reply interface{}) {
	crs := stream.GetContext().(*codeRepoSvc)
	repoInfo := crs.repoInfo

	var addr string
	if repoInfo[0] == "github.com" {
		addr = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", repoInfo[1], repoInfo[2], repoInfo[3], msg.File)
	} else if strings.Contains(repoInfo[0], "gitlab") {
		addr = fmt.Sprintf("https://%s/%s/%s/-/raw/%s/%s", repoInfo[0], repoInfo[1], repoInfo[2], repoInfo[3], msg.File)
	} else {
		return fmt.Errorf("%s not supported", repoInfo[0])
	}

	body, err := httpGet(addr)
	if err != nil {
		return err
	}
	return body
}

var codeRepoKnownMsgs = []as.KnownMessage{
	codeRepoAddr{},
	getFileContent{},
}

func init() {
	as.RegisterType((*cmdRun)(nil))
	as.RegisterType((*cmdQuery)(nil))
	as.RegisterType([]*greVMInfo(nil))
	as.RegisterType((*cmdPatternAction)(nil))
	as.RegisterType([]*greVMIDs(nil))
	as.RegisterType((*cmdTailf)(nil))
	as.RegisterType(cmdInfo{})
	as.RegisterType(codeRepoAddr{})
	as.RegisterType(getFileContent{})
}
