package gshellos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
)

type daemon struct {
	lg *log.Logger
}

func (gd *daemon) onNewStream(ctx as.Context) {
	ctx.SetContext(gd)
}

func (gd *daemon) setupgrg(grgName, grgVer, rtPriority string, maxprocs int) (as.Connection, error) {
	if len(grgVer) == 0 {
		grgVer = version
	}

	daemonMaxprocs := runtime.GOMAXPROCS(-1)
	if maxprocs <= 0 || maxprocs > daemonMaxprocs {
		maxprocs = daemonMaxprocs
	}

	name := grgName + "." + grgVer
	opts := []as.Option{
		as.WithLogger(gd.lg),
		as.WithScope(as.ScopeOS),
	}
	c := as.NewClient(opts...).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "grg-"+name)
	if conn != nil {
		return conn, nil
	}
	if grgVer != version {
		return nil, fmt.Errorf("running GRG version %s not found", grgVer)
	}

	args := "-wd " + workDir + " -loglevel " + loglevel + " __start " + "-group " + name
	if os.Args[0] == "gshell.tester" {
		args = "-test.run ^TestRunMain$ -test.coverprofile=.test/l2_grg" + name + genID(3) + ".cov -- " + args
	}
	exe := os.Args[0]
	if len(rtPriority) != 0 {
		args = rtPriority + " " + exe + " " + args
		exe = "chrt"
	}
	cmd := exec.Command(exe, strings.Split(args, " ")...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOMAXPROCS=%d", maxprocs))
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	gd.lg.Debugln("starting grg:", cmd.String())

	err := cmd.Start()
	if err != nil {
		gd.lg.Errorf("start cmd %s failed: %w", cmd.String(), err)
		return nil, err
	}
	go func() {
		cmderr := cmd.Wait()
		if cmderr != nil {
			err = errors.New(buf.String())
			gd.lg.Errorf("cmd: %s exited with error: %v, output: %v", cmd.String(), cmderr, buf.String())
		} else {
			gd.lg.Infof("cmd: %s exited, output: %v", cmd.String(), buf.String())
		}
	}()

	c.SetDiscoverTimeout(3)
	conn = <-c.Discover(godevsigPublisher, "grg-"+name)
	if conn != nil {
		return conn, nil
	}
	if err == nil {
		err = ErrBrokenGRG
	}
	return nil, err
}

type cmdKill struct {
	GRGNames []string
	GRGVer   string
	Force    bool
}

func (msg *cmdKill) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdKill: %v", msg)

	grgVer := msg.GRGVer
	if len(grgVer) == 0 {
		grgVer = version
	}

	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	var b strings.Builder
	for _, grg := range msg.GRGNames {
		connChan := c.Discover(godevsigPublisher, "grg-"+grg+"."+grgVer)
		for conn := range connChan {
			var pInfo processInfo
			conn.SetRecvTimeout(time.Second)
			if err := conn.SendRecv(getProcessInfo{}, &pInfo); err != nil {
				gd.lg.Warnf("get info for %s failed: %v", grg, err)
			}
			if pInfo.pid != 0 && (pInfo.runningCnt == 0 || msg.Force) {
				if process, err := os.FindProcess(pInfo.pid); err != nil {
					gd.lg.Warnf("pid of %s not found: %v", pInfo.grgName, err)
				} else if err := process.Signal(syscall.SIGINT); err != nil {
					gd.lg.Warnf("kill %s failed: %v", pInfo.grgName, err)
				} else {
					fmt.Fprintf(&b, "%s ", pInfo.grgName)
				}
			}
			conn.Close()
		}
	}
	if len(b.String()) == 0 {
		fmt.Fprintf(&b, "none ")
	}
	fmt.Fprintf(&b, "killed")
	return b.String()
}

type cmdRun struct {
	grgCmdRun
	GRGName    string
	GRGVer     string
	RtPriority string
	Maxprocs   int
}

func (msg *cmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdRun: file %v, args %v, interactive %v", msg.File, msg.Args, msg.Interactive)

	conn, err := gd.setupgrg(msg.GRGName, msg.GRGVer, msg.RtPriority, msg.Maxprocs)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Send(&msg.grgCmdRun); err != nil {
		return err
	}

	if !msg.Interactive {
		var greid string
		if err := conn.Recv(&greid); err != nil {
			return err
		}
		return greid
	}
	client := as.NewStreamIO(stream)
	grg := as.NewStreamIO(conn)
	gd.lg.Debugln("enter interactive io")
	done := make(chan struct{}, 1)
	go func() { io.Copy(grg, client); done <- struct{}{} }()
	go func() { io.Copy(client, grg); done <- struct{}{} }()
	<-done
	gd.lg.Debugln("exit interactive io")

	return io.EOF
}

type cmdQuery struct {
	GRGName   string
	IDPattern []string
}

func (msg *cmdQuery) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdQuery: %v", msg)

	var ggis []*grgGREInfo
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	connChan := c.Discover(godevsigPublisher, "grg-"+msg.GRGName)
	for conn := range connChan {
		var ggi *grgGREInfo
		conn.SetRecvTimeout(time.Second)
		if err := conn.SendRecv(&grgCmdQuery{msg.IDPattern}, &ggi); err != nil {
			gd.lg.Warnf("cmdQuery: send recv error: %v", err)
		}
		if ggi != nil {
			for _, grei := range ggi.GREInfos {
				if grei.Stat != "exited" {
					grei.EndTime = time.Now()
				}
			}
			ggis = append(ggis, ggi)
		}
		conn.Close()
	}
	return ggis
}

type cmdPatternAction struct {
	GRGName   string
	IDPattern []string
	Cmd       string
}

type grgGREIDs struct {
	//Name  string
	GREIDs []string
}

func (msg *cmdPatternAction) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdPatternAction: %v", msg)

	var ggreids []*grgGREIDs
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	connChan := c.Discover(godevsigPublisher, "grg-"+msg.GRGName)
	for conn := range connChan {
		var greids []string
		conn.SetRecvTimeout(time.Second)
		if err := conn.SendRecv(&grgCmdPatternAction{msg.IDPattern, msg.Cmd}, &greids); err != nil {
			gd.lg.Warnf("cmdPatternAction: send recv error: %v", err)
		}
		if greids != nil {
			ggreids = append(ggreids, &grgGREIDs{GREIDs: greids})
		}
		conn.Close()
	}
	return ggreids
}

type cmdLog struct {
	Target string
	Follow bool
}

func (msg *cmdLog) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	var file string
	switch msg.Target {
	case "daemon":
		file = workDir + "/logs/daemon.log"
	case "grg":
		file = workDir + "/logs/grg.log"
	default:
		file = workDir + "/logs/" + msg.Target
	}

	if msg.Follow {
		clientIO := as.NewStreamIO(stream)
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintln(clientIO, msg.Target+" not found")
			return
		}
		defer f.Close()
		io.Copy(clientIO, endlessReader{f})
		gd.lg.Debugln("cmdLog: done")
		clientIO.Close()
	} else {
		buf, err := os.ReadFile(file)
		if err != nil {
			return errors.New(msg.Target + " not found")
		}
		if err := stream.Send(buf); err != nil {
			return err
		}
	}

	return
}

type cmdInfo struct{}

func (msg cmdInfo) Handle(stream as.ContextStream) (reply interface{}) {
	var b strings.Builder
	fmt.Fprintf(&b, "Version: %s\n", version)
	fmt.Fprintf(&b, "Build tags: %s\n", buildTags)
	fmt.Fprintf(&b, "Commit: %s\n", commitRev)

	return b.String()
}

var daemonKnownMsgs = []as.KnownMessage{
	(*cmdKill)(nil),
	(*cmdRun)(nil),
	(*cmdQuery)(nil),
	(*cmdPatternAction)(nil),
	(*cmdLog)(nil),
	cmdInfo{},
}

var httpGet func(url string) ([]byte, error)

type updater struct {
	urlFmt string
	lg     *log.Logger
}

type gshellBin struct {
	bin []byte
	md5 string
}

// reply with *gshellBin
type tryUpdate struct {
	revInuse string
	arch     string
}

func (msg tryUpdate) Handle(stream as.ContextStream) (reply interface{}) {
	updtr := stream.GetContext().(*updater)
	updtr.lg.Debugf("tryUpdate: %v", msg)

	rev, err := httpGet(fmt.Sprintf(updtr.urlFmt, "rev"))
	if err != nil {
		return err
	}
	revNew := strings.TrimSpace(string(rev))
	updtr.lg.Debugf("tryUpdate rev: %s", revNew)
	if revNew != commitRev { // check root registry rev
		// not update other gshell daemons if root registry is not the latest
		if stream.GetNetconn().LocalAddr().Network() != "chan" {
			return ErrNoUpdate
		}
	}
	if revNew == msg.revInuse {
		return ErrNoUpdate
	}

	checksum, err := httpGet(fmt.Sprintf(updtr.urlFmt, "md5sum"))
	if err != nil {
		return err
	}

	var md5 string
	b := bytes.NewBuffer(checksum)
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, msg.arch) {
			md5 = strings.Split(line, " ")[0]
			break
		}
	}
	if len(md5) == 0 {
		return fmt.Errorf("arch %s not supported", msg.arch)
	}

	bin, err := httpGet(fmt.Sprintf(updtr.urlFmt, "gshell."+msg.arch))
	if err != nil {
		return err
	}

	return &gshellBin{bin, md5}
}

var updaterKnownMsgs = []as.KnownMessage{
	tryUpdate{},
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
	as.RegisterType((*cmdKill)(nil))
	as.RegisterType((*cmdRun)(nil))
	as.RegisterType((*cmdQuery)(nil))
	as.RegisterType([]*grgGREInfo(nil))
	as.RegisterType((*cmdPatternAction)(nil))
	as.RegisterType([]*grgGREIDs(nil))
	as.RegisterType((*cmdLog)(nil))
	as.RegisterType(cmdInfo{})
	as.RegisterType(codeRepoAddr{})
	as.RegisterType(getFileContent{})
	as.RegisterType(tryUpdate{})
	as.RegisterType((*gshellBin)(nil))
}
