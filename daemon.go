package gshellos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/glib/sys/shell"
)

type daemon struct {
	lg      *log.Logger
	workDir string
}

// some grg processes were killed by oom or unexpected operations,
// grgRestarter restarts those killed grg.
func (gd *daemon) grgRestarter() {
	for {
		time.Sleep(time.Second)
		grgs, err := filepath.Glob(gd.workDir + "/status/grg-*")
		if err != nil {
			gd.lg.Warnln(err)
			continue
		}
		for _, grgStatDir := range grgs {
			func() {
				lockFile := filepath.Join(grgStatDir, ".lock")
				f, err := os.Open(lockFile)
				if err != nil {
					gd.lg.Warnln(err)
					return
				}
				defer f.Close()

				if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
					gd.lg.Debugf("grg status %s locked, grg running", grgStatDir)
					return
				}
				gd.lg.Infof("grg status %s unlocked, grg died abnormally", grgStatDir)
				syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				strs := strings.Split(filepath.Base(grgStatDir), "-")
				if len(strs) != 4 {
					gd.lg.Warnf("grg dir name format incompatible: %v", strs)
					return
				}
				grgName := strs[1]
				rtprio, _ := strconv.Atoi(strs[2])
				maxprocs, _ := strconv.Atoi(strs[3])
				conn, err := gd.setupgrg(grgName, rtprio, maxprocs)
				if err != nil {
					gd.lg.Errorf("restart grg %s failed with error: %v", grgName, err)
					return
				}
				gd.lg.Infof("grg %s restarted", grgName)
				conn.Close()
			}()
		}
	}
}

func (gd *daemon) onNewStream(ctx as.Context) {
	ctx.SetContext(gd)
}

func (gd *daemon) setupgrg(grgName string, rtPriority int, maxprocs int) (as.Connection, error) {
	if !strings.Contains(grgName, "-") {
		grgName = grgName + "-" + version
	}
	if rtPriority < 0 {
		rtPriority = 0
	}
	if maxprocs < 0 {
		maxprocs = 0
	}

	daemonMaxprocs := runtime.GOMAXPROCS(-1)
	if maxprocs > daemonMaxprocs {
		maxprocs = daemonMaxprocs
	}

	opts := []as.Option{
		as.WithLogger(gd.lg),
		as.WithScope(as.ScopeOS),
	}
	c := as.NewClient(opts...).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "grg-"+grgName)
	if conn != nil {
		return conn, nil
	}

	if grgVer := strings.Split(grgName, "-")[1]; grgVer != version {
		return nil, fmt.Errorf("running GRG version %s not found", grgVer)
	}

	runGrg := func(exe, args string) error {
		cmd := exec.Command(exe, strings.Split(args, " ")...)
		if maxprocs != 0 {
			cmd.Env = append(os.Environ(), fmt.Sprintf("GOMAXPROCS=%d", maxprocs))
		}
		buf := &bytes.Buffer{}
		cmd.Stdout = buf
		cmd.Stderr = buf
		gd.lg.Debugln("starting grg:", cmd.String())

		if err := cmd.Start(); err != nil {
			gd.lg.Errorf("start cmd %s failed: %w", cmd.String(), err)
			return err
		}

		go func() {
			cmderr := cmd.Wait()
			if cmderr != nil {
				gd.lg.Warnf("cmd: %s exited with error: %v, output: %v", cmd.String(), cmderr, buf.String())
			} else {
				gd.lg.Infof("cmd: %s exited, output: %v", cmd.String(), buf.String())
			}
		}()

		return nil
	}

	args := fmt.Sprintf("-loglevel %s __start -group %s -wd %s", loglevel, grgName, gd.workDir)
	if os.Args[0] == "gshell.tester" {
		args = "-test.run ^TestRunMain$ -test.coverprofile=.test/l2_grg" + grgName + genID(3) + ".cov -- " + args
	}
	exe := os.Args[0]

	testChrt := func() bool {
		if err := exec.Command("chrt", strconv.Itoa(rtPriority), "true").Run(); err != nil {
			gd.lg.Infof("chrt with priority %s not working", rtPriority)
			return false
		}
		return true
	}
	if rtPriority != 0 && testChrt() {
		args = strconv.Itoa(rtPriority) + " " + exe + " " + args
		exe = "chrt"
	}
	if err := runGrg(exe, args); err != nil {
		return nil, err
	}

	c.SetDiscoverTimeout(3)
	conn = <-c.Discover(godevsigPublisher, "grg-"+grgName)
	if conn != nil {
		return conn, nil
	}
	return nil, ErrBrokenGRG
}

type cmdKill struct {
	GRGNames []string
	Force    bool
}

func (gd *daemon) doKill(msg *cmdKill) string {
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	var b strings.Builder
	var killingList []*processInfo
	for _, grg := range msg.GRGNames {
		connChan := c.Discover(godevsigPublisher, "grg-"+grg)
		for conn := range connChan {
			func() {
				defer conn.Close()
				conn.SetRecvTimeout(time.Second)
				var pInfo processInfo
				if err := conn.SendRecv(grgCmdKill{}, &pInfo); err != nil {
					gd.lg.Warnf("grgCmdKill for %s failed: %v", grg, err)
					return
				}
				if !pInfo.killing && msg.Force && pInfo.pid != 0 {
					process, err := os.FindProcess(pInfo.pid)
					if err != nil {
						gd.lg.Warnf("pid of %s not found: %v", pInfo.name, err)
						return
					}
					// prevent grgRestarter keeps restarting the grg
					os.RemoveAll(pInfo.statDir)
					if err := process.Signal(syscall.SIGKILL); err != nil {
						gd.lg.Warnf("kill %s failed: %v", pInfo.name, err)
						return
					}
					pInfo.killing = true
				}
				if pInfo.killing {
					killingList = append(killingList, &pInfo)
				}
			}()
		}
	}

	processExists := func(pid int) bool {
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return false
		}
		return true
	}

	checkDone := func() bool {
		for i, pInfo := range killingList {
			if pInfo == nil {
				continue
			}
			if !processExists(pInfo.pid) {
				killingList[i] = nil // the pid exited
				fmt.Fprintf(&b, "%s ", pInfo.name)
			}
		}
		for _, pInfo := range killingList {
			if pInfo != nil {
				return false
			}
		}
		return true
	}
	forceKill := func() {
		for _, pInfo := range killingList {
			if pInfo == nil {
				continue
			}
			// prevent grgRestarter keeps restarting the grg
			os.RemoveAll(pInfo.statDir)
			if process, err := os.FindProcess(pInfo.pid); err == nil {
				process.Signal(syscall.SIGKILL)
			}
		}
	}

	timeout := false
	time.AfterFunc(3*time.Second, func() { timeout = true })
	for {
		if timeout {
			forceKill()
		}
		if checkDone() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(b.String()) == 0 {
		fmt.Fprintf(&b, "none ")
	}

	fmt.Fprintf(&b, "killed")
	return b.String()
}

func (msg *cmdKill) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdKill: %v", msg)

	return gd.doKill(msg)
}

type cmdRun struct {
	grgCmdRun
	GRGName    string
	RtPriority int
	Maxprocs   int
}

func (msg *cmdRun) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdRun: args %v, interactive %v", msg.Args, msg.Interactive)

	conn, err := gd.setupgrg(msg.GRGName, msg.RtPriority, msg.Maxprocs)
	if err != nil {
		return err
	}
	defer conn.Close()

	killgrg := func() {
		conn.SendRecv(grgCmdKill{}, nil)
	}

	if err := conn.Send(&msg.grgCmdRun); err != nil {
		return err
	}

	if !msg.Interactive {
		var greid string
		if err := conn.Recv(&greid); err != nil {
			killgrg()
			return err
		}
		return greid
	}
	client := as.NewStreamIO(stream)
	grg := as.NewStreamIO(conn)
	gd.lg.Debugln("enter interactive io")
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(client, grg)
		if err != nil {
			killgrg()
		}
		done <- err
	}()
	go func() { io.Copy(grg, client); done <- nil }()
	err = <-done
	gd.lg.Debugln("exit interactive io")
	if err != nil {
		return err
	}
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
		file = gd.workDir + "/logs/daemon.log"
	case "grg":
		file = gd.workDir + "/logs/grg.log"
	default:
		file = gd.workDir + "/logs/" + msg.Target
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

type joblist struct {
	GRGs []grgJoblist
}

// reply joblist{}
type cmdJoblistSave struct {
	Tiny bool
}

func (msg cmdJoblistSave) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdJoblistSave: %v", msg)

	jlist := &joblist{}
	c := as.NewClient(as.WithLogger(gd.lg), as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
	connChan := c.Discover(godevsigPublisher, "grg-*")
	for conn := range connChan {
		var grgjl grgJoblist
		conn.SetRecvTimeout(time.Second)
		if err := conn.SendRecv(grgCmdJoblist{msg.Tiny}, &grgjl); err != nil {
			gd.lg.Warnf("cmdJoblistSave: send recv error: %v", err)
		} else {
			grgjl.Name = strings.Split(grgjl.Name, "-")[0]
			jlist.GRGs = append(jlist.GRGs, grgjl)
		}
		conn.Close()
	}
	return jlist
}

// reply OK or error
type cmdJoblistLoad struct {
	joblist
	requestedBy string // by which provider ID
}

func (msg *cmdJoblistLoad) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	gd.lg.Debugf("handle cmdJoblistLoad: %v", msg)

	out := gd.doKill(&cmdKill{GRGNames: []string{"*"}, Force: true})
	gd.lg.Infoln("kill all GRGs:", out)

	var wg sync.WaitGroup
	errChan := make(chan error, len(msg.GRGs))
	for _, grgjl := range msg.GRGs {
		grgjl := grgjl
		wg.Add(1)
		go func() {
			defer wg.Done()
			grgconn, err := gd.setupgrg(grgjl.Name, grgjl.RtPriority, grgjl.Maxprocs)
			if err != nil {
				gd.lg.Errorf("load grg %s failed with error: %v", grgjl.Name, err)
				errChan <- err
				return
			}
			defer grgconn.Close()

			for _, job := range grgjl.Jobs {
				runMsg := &grgCmdRun{
					JobCmd:      job.JobCmd,
					Interactive: false,
					RequestedBy: msg.requestedBy,
				}

				if err := grgconn.SendRecv(runMsg, nil); err != nil {
					gd.lg.Errorln(err)
					errChan <- err
					return
				}
			}
			gd.lg.Infof("grg %s loaded", grgjl.Name)
		}()
	}

	wg.Wait()
	if len(errChan) == 0 {
		return as.OK
	}
	close(errChan)
	err := errors.New("load joblist error")
	for e := range errChan {
		err = fmt.Errorf("%v, %v", err, e)
	}
	return err
}

type codeRepoAddrByNode struct{}

func (msg codeRepoAddrByNode) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	c := as.NewClient(as.WithLogger(gd.lg)).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "codeRepo")
	if conn == nil {
		return as.ErrServiceNotFound(godevsigPublisher, "codeRepo")
	}
	defer conn.Close()
	addr := "NA"
	if err := conn.SendRecv(codeRepoAddr{}, &addr); err != nil {
		return err
	}
	return addr
}

type codeRepoListByNode struct {
	codeRepoList
}

func (msg codeRepoListByNode) Handle(stream as.ContextStream) (reply interface{}) {
	gd := stream.GetContext().(*daemon)
	c := as.NewClient(as.WithLogger(gd.lg)).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "codeRepo")
	if conn == nil {
		return as.ErrServiceNotFound(godevsigPublisher, "codeRepo")
	}
	defer conn.Close()
	var entries []dirEntry
	if err := conn.SendRecv(msg.codeRepoList, &entries); err != nil {
		return err
	}
	return entries
}

var daemonKnownMsgs = []as.KnownMessage{
	(*cmdKill)(nil),
	(*cmdRun)(nil),
	(*cmdQuery)(nil),
	(*cmdPatternAction)(nil),
	(*cmdLog)(nil),
	cmdInfo{},
	cmdJoblistSave{},
	(*cmdJoblistLoad)(nil),
	codeRepoAddrByNode{},
	codeRepoListByNode{},
}

type updater struct {
	url string
	lg  *log.Logger
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

	rev, err := httpOp.readFile(updtr.url + "/rev")
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

	checksum, err := httpOp.readFile(updtr.url + "/md5sum")
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

	bin, err := httpOp.readFile(updtr.url + "/gshell." + msg.arch)
	if err != nil {
		return err
	}

	return &gshellBin{bin, md5}
}

var updaterKnownMsgs = []as.KnownMessage{
	tryUpdate{},
}

type codeRepoSvc struct {
	localRepoPath string
	httpRepoInfo  []string // site/org/proj/branch
}

type codeRepoAddr struct{}

func (msg codeRepoAddr) Handle(stream as.ContextStream) (reply interface{}) {
	crs := stream.GetContext().(*codeRepoSvc)
	if crs.localRepoPath != "" {
		return crs.localRepoPath
	}
	return strings.Join(crs.httpRepoInfo[:3], "/") + " " + crs.httpRepoInfo[3]
}

func goModVendor(zip []byte) ([]byte, error) {
	tmpDir, err := os.MkdirTemp(gshellTempDir, "getcode-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	if err := unzipBufferToPath(zip, tmpDir); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "vendor")); err == nil {
		return zip, nil // already included
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "mod")); err != nil {
		if out, err := shell.Run(fmt.Sprintf("cd %s; go mod init main", tmpDir)); err != nil {
			return nil, fmt.Errorf("go mod init failed with %s", out)
		}
	}
	if out, err := shell.Run(fmt.Sprintf("cd %s; go mod tidy", tmpDir)); err != nil {
		return nil, fmt.Errorf("go mod tidy failed with %s", out)
	}
	if out, err := shell.Run(fmt.Sprintf("cd %s; go mod vendor", tmpDir)); err != nil {
		return nil, fmt.Errorf("go mod vendor failed with %s", out)
	}
	zip, err = zipPathToBuffer(tmpDir)
	if err != nil {
		return nil, err
	}
	return zip, nil
}

type getCode struct {
	PathFile   string
	AutoImport bool
}

func (msg getCode) Handle(stream as.ContextStream) (reply interface{}) {
	crs := stream.GetContext().(*codeRepoSvc)
	HTTPAddr := ""
	if strings.HasPrefix(msg.PathFile, "http://") || strings.HasPrefix(msg.PathFile, "https://") {
		HTTPAddr = msg.PathFile // raw URL
	}
	var zip []byte
	var err error
	if HTTPAddr != "" || len(crs.httpRepoInfo) != 0 {
		if httpOp == nil {
			return errors.New("No http functionality, check the build tags")
		}

		if HTTPAddr == "" {
			domain, owner, repo, branch := crs.httpRepoInfo[0], crs.httpRepoInfo[1], crs.httpRepoInfo[2], crs.httpRepoInfo[3]
			if strings.Contains(domain, "github") {
				HTTPAddr = fmt.Sprintf("https://%s/%s/%s/tree/%s/%s", domain, owner, repo, branch, msg.PathFile)
			} else if strings.Contains(domain, "gitlab") {
				HTTPAddr = fmt.Sprintf("https://%s/%s/%s/-/tree/%s/%s", domain, owner, repo, branch, msg.PathFile)
			} else {
				return fmt.Errorf("%s not supported", domain)
			}
		}

		zip, err = httpOp.getArchive(HTTPAddr)
	} else if crs.localRepoPath != "" { // local dir
		zip, err = zipPathToBuffer(filepath.Join(crs.localRepoPath, msg.PathFile))
	} else {
		err = errors.New("code repo not configured")
	}

	if err != nil {
		return err
	}
	if !msg.AutoImport {
		return zip
	}

	zip, err = goModVendor(zip)
	if err != nil {
		return err
	}
	return zip
}

// reply with []dirEntry
type codeRepoList struct {
	path string
}

type dirEntry struct {
	name  string
	isDir bool
}

func (msg codeRepoList) Handle(stream as.ContextStream) (reply interface{}) {
	crs := stream.GetContext().(*codeRepoSvc)
	var entries []dirEntry
	// local dir
	if crs.localRepoPath != "" {
		path := filepath.Join(crs.localRepoPath, msg.path)
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			return []dirEntry{{filepath.Base(path), false}}
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				entries = append(entries, dirEntry{file.Name(), true})
			} else {
				entries = append(entries, dirEntry{file.Name(), false})
			}
		}
		return entries
	}
	domain, owner, repo, branch := crs.httpRepoInfo[0], crs.httpRepoInfo[1], crs.httpRepoInfo[2], crs.httpRepoInfo[3]

	var addr string
	if strings.HasPrefix(msg.path, "http://") || strings.HasPrefix(msg.path, "https://") {
		addr = msg.path // raw URL
	} else if strings.Contains(domain, "github") {
		addr = fmt.Sprintf("https://%s/%s/%s/tree/%s/%s", domain, owner, repo, branch, msg.path)
	} else if strings.Contains(domain, "gitlab") {
		addr = fmt.Sprintf("https://%s/%s/%s/-/tree/%s/%s", domain, owner, repo, branch, msg.path)
	} else {
		return fmt.Errorf("%s not supported", domain)
	}

	hfis, err := httpOp.list(addr)
	if err != nil {
		return err
	}
	for _, hfi := range hfis {
		entries = append(entries, dirEntry{hfi.name, hfi.isDir})
	}
	return entries
}

var codeRepoKnownMsgs = []as.KnownMessage{
	codeRepoAddr{},
	getCode{},
	codeRepoList{},
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
	as.RegisterType(cmdJoblistSave{})
	as.RegisterType((*joblist)(nil))
	as.RegisterType(codeRepoAddrByNode{})
	as.RegisterType(codeRepoListByNode{})
	as.RegisterType(codeRepoAddr{})
	as.RegisterType((*cmdJoblistLoad)(nil))
	as.RegisterType(getCode{})
	as.RegisterType(codeRepoList{})
	as.RegisterType([]dirEntry(nil))
	as.RegisterType(tryUpdate{})
	as.RegisterType((*gshellBin)(nil))
}
