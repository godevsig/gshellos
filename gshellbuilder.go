package gshellos

import (
	"context"
	"crypto/md5"
	_ "embed" // go embed
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/glib/sys/shell"
	"github.com/traefik/yaegi/interp"
)

//go:embed bin/version
var version string

//go:embed bin/buildtag
var buildTags string

//go:embed bin/buildtime
var buildTime string

const (
	defaultWorkDir    = "/var/tmp/gshell"
	godevsigPublisher = "godevsig"
	defaultLibDir     = "/usr/lib/gshell"
)

var (
	loglevel     = "info"
	providerID   = "self"
	pluginDir    = defaultLibDir + "/plugins"
	debugService func(lg *log.Logger)
)

func init() {
	if len(version) == 0 {
		version = "devel"
	}
	if len(buildTags) == 0 {
		buildTags = "unknown"
	}
	if len(buildTime) == 0 {
		buildTime = "unknown"
	}
}

func getSelfID() string {
	selfID, err := as.GetSelfProviderID()
	if err != nil {
		selfID = "NA"
	}
	return selfID
}

func trimName(name string, size int) string {
	if len(name) > size {
		name = name[:size-3] + "..."
	}
	return name
}

type subCmd struct {
	*flag.FlagSet
	action func() error
}

var cmds []subCmd

func newLogger(logStream *log.Stream, loggerName string) *log.Logger {
	level := log.Linfo
	switch loglevel {
	case "debug":
		level = log.Ldebug
		logStream.SetFlag(log.Lfileline)
	case "warn":
		level = log.Lwarn
	case "error":
		level = log.Lerror
	}
	logStream.SetLoglevel("*", level)
	return logStream.NewLogger(loggerName, log.Linfo)
}

func newCmd(name, usage string, helps ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", name, usage)
	for _, line := range helps {
		fmt.Fprintf(&b, "\n\t%s", line)
	}
	return b.String()
}

var updateInterval = "600"

func addDaemonCmd() {
	cmd := flag.NewFlagSet(newCmd("daemon", "[options]", "Start local gshell daemon"), flag.ExitOnError)
	nodeProviderID := cmd.String("id", "auto", "specify self node providerID")
	workDir := cmd.String("wd", defaultWorkDir, "set working directory")
	namedUDS := cmd.Bool("namedsocket", false, "use named Unix Domain Socket for the daemon")
	rootRegistry := cmd.Bool("root", false, "enable root registry service")
	invisible := cmd.Bool("invisible", false, "make gshell daemon invisible in gshell service network")
	registryAddr := cmd.String("registry", "", "root registry address in host:port format")
	lanBroadcastPort := cmd.String("bcast", "", "broadcast port for LAN")
	codeRepo := cmd.String("repo", "", "code repo local path or https address in format site/org/proj/branch")
	updateURL := cmd.String("update", "", "url of artifacts to update gshell, require -root")
	clean := cmd.Bool("clean", false, "clean status/log/code files from old runs before starting the new run")

	daemonRunning := func() bool {
		c := as.NewClient(as.WithScope(as.ScopeOS)).SetDiscoverTimeout(0)
		conn := <-c.Discover(godevsigPublisher, "gshellDaemon")
		if conn == nil {
			return false
		}
		conn.Close()
		return true
	}

	action := func() error {
		if providerID != "self" {
			return errors.New("command does not run on remote node")
		}

		if daemonRunning() {
			return errors.New("detected a running gshell daemon on local node")
		}

		workDir := *workDir
		if *clean {
			killGrgByName("*")
			os.RemoveAll(workDir)
			os.RemoveAll(gshellTempDir)
		}

		if err := os.MkdirAll(gshellTempDir, 0755); err != nil {
			return err
		}

		if err := os.MkdirAll(workDir+"/logs", 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(workDir+"/status", 0755); err != nil {
			return err
		}

		logStream := log.NewStream("daemon")
		if err := logStream.SetOutput("file:" + workDir + "/logs/daemon.log"); err != nil {
			return err
		}
		lg := newLogger(logStream, "daemon")
		lg.Infof("daemon version: %s", version)
		if *clean {
			lg.Infof("work dir and temp dir recreated")
		}

		cmdArgs := os.Args
		scope := as.ScopeAll
		if len(*registryAddr) == 0 {
			scope &= ^as.ScopeWAN // not ScopeWAN
		}
		if len(*lanBroadcastPort) == 0 {
			scope &= ^as.ScopeLAN // not ScopeLAN
		}
		updateURL := *updateURL
		if len(updateURL) != 0 || *rootRegistry {
			if scope&as.ScopeWAN != as.ScopeWAN {
				return errors.New("root registry address not set")
			}
		}

		codeRepo := *codeRepo
		crs := &codeRepoSvc{}
		if len(codeRepo) != 0 {
			fi, err := os.Stat(codeRepo)
			if err != nil || !fi.Mode().IsDir() {
				crs.httpRepoInfo = strings.Split(codeRepo, "/")
				if len(crs.httpRepoInfo) != 4 {
					return errors.New("wrong repo format")
				}
				if httpOp == nil {
					return errors.New("http feature not enabled, check build tags")
				}
			} else {
				crs.localRepoPath, _ = filepath.Abs(codeRepo)
			}
		}

		euid := os.Geteuid()
		if err := syscall.Setreuid(euid, euid); err != nil {
			return err
		}
		egid := os.Getegid()
		if err := syscall.Setregid(egid, egid); err != nil {
			return err
		}

		opts := []as.Option{
			as.WithScope(scope),
			as.WithLogger(lg),
		}
		registryAddr := *registryAddr
		if len(registryAddr) != 0 {
			opts = append(opts, as.WithRegistryAddr(registryAddr))
		}
		nodeProviderID := *nodeProviderID
		if nodeProviderID != "auto" {
			opts = append(opts, as.WithProviderID(nodeProviderID))
		}

		s := as.NewServer(opts...).
			SetPublisher(godevsigPublisher).
			SetScaleFactors(4, 0, 0).
			EnableServiceLister()
		defer s.Close()

		if len(*lanBroadcastPort) != 0 {
			s.SetBroadcastPort(*lanBroadcastPort)
		}
		if !*invisible {
			s.EnableMessageTracer()
			if scope&as.ScopeNetwork != 0 {
				s.EnableAutoReverseProxy()
			}
		}

		if *rootRegistry {
			s.EnableRootRegistry()
			s.EnableIPObserver()

			if len(updateURL) != 0 {
				if httpOp == nil {
					return errors.New("http feature not enabled, check build tags")
				}
				updateURL = strings.TrimSuffix(updateURL, "/")
				updtr := &updater{url: updateURL, lg: lg}
				if err := s.Publish("updater",
					updaterKnownMsgs,
					as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(updtr) }),
				); err != nil {
					return err
				}
			}
		}

		if len(crs.localRepoPath) != 0 || len(crs.httpRepoInfo) != 0 {
			scope := scope
			if len(crs.localRepoPath) != 0 {
				scope &= ^as.ScopeWAN // not ScopeWAN
				scope &= ^as.ScopeLAN // not ScopeLAN
			}
			if err := s.PublishIn(scope, "codeRepo",
				codeRepoKnownMsgs,
				as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(crs) }),
			); err != nil {
				return err
			}
		}

		var updateChan chan struct{}
		go func() {
			if _, has := os.LookupEnv("GSHELL_NOUPDATE"); has {
				lg.Infoln("no auto update")
				return
			}
			i, _ := strconv.Atoi(updateInterval)
			lg.Debugf("updater interval: %d", i)
			exe, err := os.Executable()
			if err != nil {
				lg.Warnf("executable path error: %s", err)
				return
			}
			lg.Debugf("executable path: %s", exe)
			for {
				time.Sleep(time.Duration(i) * time.Second)
				c := as.NewClient(as.WithLogger(lg)).SetDiscoverTimeout(0)
				conn := <-c.Discover(godevsigPublisher, "updater")
				if conn == nil {
					continue
				}
				var gshellbin *gshellBin
				lg.Debugln("trying to update")
				err := conn.SendRecv(tryUpdate{revInuse: version, arch: runtime.GOARCH}, &gshellbin)
				conn.Close()
				lg.Debugln("updater returned with", err)
				if err != nil {
					if strings.Contains(err.Error(), ErrNoUpdate.Error()) {
						lg.Debugln(ErrNoUpdate)
					} else {
						lg.Warnf("get gshell bin error: %v", err)
					}
					continue
				}
				if fmt.Sprintf("%x", md5.Sum(gshellbin.bin)) != gshellbin.md5 {
					lg.Warnf("gshell new version md5 mismatch")
					continue
				}
				newFile := workDir + "/gshell.updating"
				if err := os.WriteFile(newFile, gshellbin.bin, 0755|fs.ModeSetuid|fs.ModeSetgid); err != nil {
					lg.Warnf("create gshell new version failed")
					continue
				}

				lg.Debugln(shell.Run("ls -lh " + newFile))
				lg.Infof("updating gshell version...")

				if err := os.Rename(newFile, exe); err != nil {
					lg.Infof("failed to rename new gshell to %s: %s", exe, err)
					output, _ := shell.Run("mv -f " + newFile + " " + exe)
					if len(output) != 0 {
						lg.Warnf("failed to mv new gshell to %s: %s", exe, output)
						continue
					}
				}
				if output, _ := shell.Run("chmod ugo+s " + exe); len(output) != 0 {
					lg.Warnf("failed to set gshell bin permission : %s", output)
					continue
				}

				updateChan = make(chan struct{})
				s.CloseWait()
				cmd := cmdArgs[0]
				args := cmdArgs[1:]
				if cmdArgs[0] == "gshell.tester" {
					coverageOpt := "-test.coverprofile=.test/l2_gshelld" + genID(3) + ".cov"
					args = append([]string{"-test.run", "^TestRunMain$", coverageOpt, "--"}, args...)
				}
				var newArgs []string
				for _, s := range args {
					if !strings.Contains(s, "-clean") {
						newArgs = append(newArgs, s)
					}
				}
				if err := exec.Command(cmd, newArgs...).Start(); err != nil {
					lg.Errorf("start new gshell failed: %v", err)
				} else {
					lg.Infof("new version gshell started")
				}
				close(updateChan)
				return
			}
		}()

		gd := &daemon{
			lg:      lg,
			workDir: workDir,
		}
		visibleScope := scope
		if *invisible {
			visibleScope = as.ScopeProcess | as.ScopeOS
		}

		svcOpts := []as.ServiceOption{
			as.OnNewStreamFunc(gd.onNewStream),
		}
		if *namedUDS {
			svcOpts = append(svcOpts, as.UseNamedUDS())
		}

		if err := s.PublishIn(visibleScope, "gshellDaemon",
			daemonKnownMsgs,
			svcOpts...,
		); err != nil {
			return err
		}
		if debugService != nil {
			go debugService(lg)
		}

		go gd.grgRestarter()
		err := s.Serve()
		if updateChan != nil {
			<-updateChan
		}
		return err
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addListCmd() {
	cmd := flag.NewFlagSet(newCmd("list", "[options]", "List services in all scopes"), flag.ExitOnError)
	verbose := cmd.Bool("v", false, "show verbose info")
	publisher := cmd.String("p", "*", "publisher name, can be wildcard")
	service := cmd.String("s", "*", "service name, can be wildcard")

	action := func() error {
		if providerID != "self" {
			return errors.New("command does not run on remote node")
		}
		opts := []as.Option{
			as.WithScope(as.ScopeProcess | as.ScopeOS),
			as.WithLogger(newLogger(log.DefaultStream, "main")),
		}

		c := as.NewClient(opts...).SetDiscoverTimeout(0)
		conn := <-c.Discover(as.BuiltinPublisher, as.SrvServiceLister)
		if conn == nil {
			return as.ErrServiceNotFound(as.BuiltinPublisher, as.SrvServiceLister)
		}
		defer conn.Close()

		msg := as.ListService{TargetScope: as.ScopeAll, Publisher: *publisher, Service: *service}
		var scopes [4][]*as.ServiceInfo
		if err := conn.SendRecv(&msg, &scopes); err != nil {
			return err
		}
		selfID := getSelfID()
		if *verbose {
			for _, services := range scopes {
				for _, svc := range services {
					if svc.ProviderID == selfID {
						svc.ProviderID = "self"
					}
					fmt.Printf("PUBLISHER: %s\n", svc.Publisher)
					fmt.Printf("SERVICE  : %s\n", svc.Service)
					fmt.Printf("PROVIDER : %s\n", svc.ProviderID)
					addr := svc.Addr
					if addr[len(addr)-1] == 'P' {
						addr = addr[:len(addr)-1] + "(proxied)"
					}
					fmt.Printf("ADDRESS  : %s\n\n", addr)
				}
			}
		} else {
			list := make(map[string]*as.Scope)
			for i, services := range scopes {
				for _, svc := range services {
					if svc.ProviderID == selfID {
						svc.ProviderID = "self"
					}
					k := svc.Publisher + "_" + svc.Service + "_" + svc.ProviderID
					p, has := list[k]
					if !has {
						v := as.Scope(0)
						p = &v
						list[k] = p
					}
					*p = *p | 1<<i
				}
			}
			names := make([]string, 0, len(list))
			for name := range list {
				names = append(names, name)
			}
			sort.Strings(names)
			fmt.Println("PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)")
			for _, svc := range names {
				p := list[svc]
				if p == nil {
					panic("nil p")
				}
				ss := strings.SplitN(svc, "_", 3)
				fmt.Printf("%-24s  %-24s  %-12s  %4b\n", trimName(ss[0], 24), trimName(ss[1], 24), trimName(ss[2], 12), *p)
			}
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addIDCmd() {
	cmd := flag.NewFlagSet(newCmd("id", "", "Print self provider ID"), flag.ExitOnError)

	action := func() error {
		if providerID != "self" {
			return errors.New("command does not run on remote node")
		}
		selfID := getSelfID()
		fmt.Println(selfID)
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addExecCmd() {
	cmd := flag.NewFlagSet(newCmd("exec", "<path[/file.go]> [args...]", "Run go file(s) in a local GRE"), flag.ExitOnError)

	action := func() error {
		if providerID != "self" {
			return errors.New("command does not run on remote node")
		}
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no path provided, see gshell exec --help")
		}
		if err := loadPlugins(pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "load plugins error: %v", err)
		}

		filePath := args[0]
		zip, err := zipPathToBuffer(filePath)
		if err != nil {
			return err
		}

		gsh, err := newShellWithCodeZip(zip)
		if err != nil {
			return err
		}
		defer gsh.close()

		if err := gsh.init(interp.Options{Args: args}); err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		return gsh.start(ctx)
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addStartCmd() {
	cmd := flag.NewFlagSet(newCmd("__start", "[options]", "Start named GRG"), flag.ExitOnError)
	workDir := cmd.String("wd", defaultWorkDir, "set working directory")
	grgName := cmd.String("group", "", "GRG name")

	getRealtimePriority := func(pid int) int {
		statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			return 0
		}
		fields := strings.Fields(string(statData))
		if len(fields) < 40 {
			return 0
		}
		priorityStr := fields[39]
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			return 0
		}
		return priority
	}

	action := func() error {
		grgName := *grgName
		if providerID != "self" {
			return errors.New("command does not run on remote node")
		}
		if len(grgName) == 0 {
			return errors.New("no GRG name, see gshell __start --help")
		}

		workDir := *workDir
		logStream := log.NewStream("grg")
		logStream.SetOutput("file:" + workDir + "/logs/grg.log")
		lg := newLogger(logStream, "grg-"+grgName)
		if err := loadPlugins(pluginDir); err != nil {
			lg.Warnf("load plugins error: %v", err)
		}

		opts := []as.Option{
			as.WithScope(as.ScopeOS),
			as.WithLogger(lg),
		}

		var serverErr error
		rtprio := getRealtimePriority(os.Getpid())
		maxProcs := 0
		if v, has := os.LookupEnv("GOMAXPROCS"); has {
			if i, err := strconv.Atoi(v); err == nil {
				maxProcs = i
			}
		}
		grgStatDir := fmt.Sprintf("%s/status/grg-%s-%d-%d", workDir, grgName, rtprio, maxProcs)
		if err := os.MkdirAll(grgStatDir, 0755); err != nil {
			return err
		}
		defer func() {
			// remove the file only if no errors
			if serverErr == nil {
				os.RemoveAll(grgStatDir)
			}
		}()
		grgLockFile := filepath.Join(grgStatDir, ".lock")
		f, err := os.OpenFile(grgLockFile, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			return err
		}
		lg.Debugf("status %s locked", grgStatDir)
		defer func() {
			syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			lg.Debugf("status %s unlocked", grgStatDir)
		}()

		s := as.NewServer(opts...).SetPublisher(godevsigPublisher)
		grg := &grg{
			processInfo: processInfo{
				name:       grgName,
				rtPriority: rtprio,
				maxProcs:   maxProcs,
				statDir:    grgStatDir,
				pid:        os.Getpid(),
			},
			workDir: workDir,
			server:  s,
			lg:      lg,
			gres:    make(map[string]*greCtl),
		}
		if err := grg.loadGREs(); err != nil {
			return err
		}

		if err := s.Publish("grg-"+grgName,
			grgKnownMsgs,
			as.OnNewStreamFunc(grg.onNewStream),
		); err != nil {
			return err
		}
		if debugService != nil {
			go debugService(lg)
		}

		serverErr = s.Serve()
		return serverErr
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func connectDaemon(providerID string, lg *log.Logger) (conn as.Connection) {
	opts := []as.Option{}
	if lg != nil {
		opts = append(opts, as.WithLogger(lg))
	}
	if providerID == "self" || providerID == getSelfID() { // local
		opts = append(opts, as.WithScope(as.ScopeOS))
		c := as.NewClient(opts...).SetDiscoverTimeout(3)
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
	} else { // remote
		opts = append(opts, as.WithScope(as.ScopeOS|as.ScopeNetwork))
		// added as.ScopeOS to cover the shared named UDS use case
		c := as.NewClient(opts...).SetDiscoverTimeout(3)
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon", providerID)
	}
	return
}

func addRepoCmd() {
	cmd := flag.NewFlagSet(newCmd("repo", "[ls [path]]", "List contents of the code repo seen on local and remote node"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		conn := connectDaemon(providerID, nil)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		localSrcDir := filepath.Join(defaultLibDir, "src")
		if len(args) == 0 {
			fmt.Println("[Local] : " + localSrcDir)
			addr := "NA"
			conn.SendRecv(codeRepoAddrByNode{}, &addr)
			fmt.Println("[Remote]: " + addr)
			return nil
		}

		if args[0] == "ls" {
			path := ""
			if len(args) >= 2 {
				path = args[1]
			}
			fmt.Println("[Local]:")
			if entries, err := os.ReadDir(filepath.Join(localSrcDir, path)); err == nil {
				for _, e := range entries {
					if e.IsDir() {
						fmt.Printf("\x1b[34m%s\x1b[0m\n", e.Name())
					} else {
						fmt.Println(e.Name())
					}
				}
			}

			fmt.Println("\n[Remote]:")
			var entries []dirEntry
			if err := conn.SendRecv(codeRepoListByNode{codeRepoList{path}}, &entries); err != nil {
				fmt.Println("NA")
				return nil
			}
			for _, e := range entries {
				if e.isDir {
					fmt.Printf("\x1b[34m%s\x1b[0m\n", e.name)
				} else {
					fmt.Println(e.name)
				}
			}
			return nil
		}
		return fmt.Errorf("unknown command %s, see gshell repo --help", args[0])
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addRunCmd() {
	cmd := flag.NewFlagSet(newCmd("run",
		"[options] <path[/file.go]> [args...]",
		"Run path[/file.go] in a new GRE in specified GRG on local/remote node.",
		"The source code is searched in below order:",
		"try local ./path[/file.go] or else",
		"try with -src specified prefix dir or else",
		"fetch the code from the remote part in `gshell repo`.",
		"Use `gshell repo ls [path]` to see available code files"),
		flag.ExitOnError)
	grgName := cmd.String("group", "", `target group name, usually specifying an existing GRG
random group name will be used if no name specified`)
	maxprocs := cmd.Int("maxprocs", 0, "set GOMAXPROCS variable")
	rtPriority := cmd.Int("rt", 0, `set the GRG to SCHED_RR min/max priority 1/99 on new GRG creation
silently ignore errors if real-time priority can not be set`)
	interactive := cmd.Bool("i", false, "enter interactive mode")
	autoRemove := cmd.Bool("rm", false, "auto-remove the GRE when it exits")
	autoRestart := cmd.Uint("restart", 0, `auto-restart the GRE on failure for at most specified times
only applicable for non-interactive mode`)
	autoImport := cmd.Bool("import", false, "auto-import dependent packages")
	srcDir := cmd.String("src", defaultLibDir+"/src/", "src prefix dir")

	action := func() error {
		grgName := *grgName
		srcDir := *srcDir
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see gshell run --help")
		}

		if len(grgName) == 0 {
			grgName = randStringRunes(6)
		} else {
			if strings.Contains(grgName, "*") {
				return errors.New("wrong use of wildcard(*), see gshell run --help")
			}
			if strings.Count(grgName, "-") > 1 {
				return errors.New("wrong group format, see gshell run --help")
			}
		}
		maxprocs := *maxprocs
		if maxprocs < 0 {
			maxprocs = 0
		}
		rtPriority := *rtPriority
		if rtPriority < 0 || rtPriority > 99 {
			return errors.New("wrong SCHED_RR priority, see man chrt")
		}

		lg := newLogger(log.DefaultStream, "main")

		if *interactive {
			*autoRestart = 0
		}

		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		jobcmd := JobCmd{
			Args:           args,
			AutoRemove:     *autoRemove,
			AutoRestartMax: *autoRestart,
		}

		// try to use local file/path if it exists
		filePath := args[0]
		if zip, err := zipPathToBuffer(filePath); err == nil {
			jobcmd.CodeZip = zip
		}

		// try to use it with -src prefix
		filePath = srcDir + args[0]
		if zip, err := zipPathToBuffer(filePath); err == nil {
			jobcmd.CodeZip = zip
		}

		selfID := getSelfID()
		cmd := cmdRun{
			grgCmdRun: grgCmdRun{
				JobCmd:      jobcmd,
				Interactive: *interactive,
				AutoImport:  *autoImport,
				RequestedBy: selfID,
			},
			GRGName:    grgName,
			RtPriority: rtPriority,
			Maxprocs:   maxprocs,
		}

		if err := conn.Send(&cmd); err != nil {
			return err
		}

		if !*interactive {
			var greid string
			if err := conn.Recv(&greid); err != nil {
				return err
			}
			fmt.Println(greid)
			return nil
		}

		ioconn := as.NewStreamIO(conn)
		lg.Debugln("enter interactive io")
		go io.Copy(ioconn, os.Stdin)
		_, err := io.Copy(os.Stdout, ioconn)
		lg.Debugln("exit interactive io")
		return err
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addKillCmd() {
	cmd := flag.NewFlagSet(newCmd("kill",
		"[options] names ...",
		"Terminate the named GRG(s) on local/remote node",
		"name supports simple wildcard(*), but must be full GRG name when force kill"),
		flag.ExitOnError)
	force := cmd.Bool("f", false, "force terminate the GRG even if there are still running GREs in it")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no GRG specified, see gshell kill --help")
		}
		force := *force
		if force {
			str := strings.Join(args, " ")
			if strings.Contains(str, "*") {
				return errors.New("wildcard not supported when force kill, see gshell kill --help")
			}
		}

		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		cmd := cmdKill{
			GRGNames: args,
			Force:    force,
		}
		var reply string
		if err := conn.SendRecv(&cmd, &reply); err != nil {
			return err
		}
		fmt.Println(reply)
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addJoblistCmd() {
	cmd := flag.NewFlagSet(newCmd("joblist",
		"[options] <save|load>",
		"Save all current jobs to file or load them to run on local/remote node"),
		flag.ExitOnError)
	file := cmd.String("file", "default.joblist.yaml", "the file save to or load from")
	tiny := cmd.Bool("tiny", false, "discard bytecode")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no subcommand provided, see gshell joblist --help")
		}
		action := args[0]

		file, err := filepath.Abs(*file)
		if err != nil {
			return err
		}

		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		tiny := *tiny
		switch action {
		case "save":
			var jlist joblist
			if err := conn.SendRecv(cmdJoblistSave{tiny}, &jlist); err != nil {
				return err
			}
			// turn bytecode to string
			for _, grg := range jlist.GRGs {
				for _, job := range grg.Jobs {
					if !tiny {
						job.CodeZipBase64 = base64.StdEncoding.EncodeToString(job.CodeZip)
					}
					job.CodeZip = nil
					job.Cmd = strings.Join(job.Args, " ")
					job.Args = nil
				}
			}
			f, err := os.Create(file)
			if err != nil {
				return err
			}
			defer f.Close()

			enc := yaml.NewEncoder(f)
			if err := enc.Encode(jlist); err != nil {
				return err
			}
			fmt.Println(file, "saved")
		case "load":
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			var jlist joblist
			dec := yaml.NewDecoder(f)
			if err := dec.Decode(&jlist); err != nil {
				return err
			}

			grgMap := make(map[string]struct{})
			// back to bytecode
			for _, grg := range jlist.GRGs {
				if len(grg.Name) == 0 {
					return fmt.Errorf("empty grg name in joblist %s", file)
				}
				if _, has := grgMap[grg.Name]; has {
					return fmt.Errorf("duplicated grg entry %s in joblist %s", grg.Name, file)
				}
				grgMap[grg.Name] = struct{}{}
				for _, job := range grg.Jobs {
					if len(job.CodeZipBase64) != 0 {
						data, err := base64.StdEncoding.DecodeString(job.CodeZipBase64)
						if err != nil {
							return fmt.Errorf("parse joblist %s with error: %v", file, err)
						}
						job.CodeZipBase64 = ""
						job.CodeZip = data
					}
					job.Args = strings.Fields(job.Cmd)
					if len(job.Args) == 0 {
						return fmt.Errorf("parse joblist %s error: empty job", file)
					}
					job.Cmd = ""
				}
			}

			selfID := getSelfID()
			if err := conn.SendRecv(&cmdJoblistLoad{jlist, selfID}, nil); err != nil {
				return err
			}
		default:
			return errors.New("wrong subcommand, see gshell joblist --help")
		}

		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPsCmd() {
	cmd := flag.NewFlagSet(newCmd("ps", "[options] [GRE IDs ...|names ...]", "Show jobs by GRE ID or name on local/remote node"), flag.ExitOnError)
	grgName := cmd.String("group", "*", "in which GRG, wildcard(*) is supported")

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		msg := cmdQuery{GRGName: *grgName, IDPattern: cmd.Args()}
		var ggis []*grgGREInfo
		if err := conn.SendRecv(&msg, &ggis); err != nil {
			return err
		}

		for _, ggi := range ggis {
			grei := ggi.GREInfos
			sort.Slice(grei, func(i int, j int) bool {
				return grei[i].StartTime.After(grei[j].StartTime)
			})
		}

		sort.Slice(ggis, func(i, j int) bool {
			if len(ggis[i].GREInfos) == 0 {
				return false
			}
			if len(ggis[j].GREInfos) == 0 {
				return true
			}
			return ggis[i].GREInfos[0].StartTime.After(ggis[j].GREInfos[0].StartTime)
		})

		if len(msg.IDPattern) != 0 { // info
			for _, ggi := range ggis {
				for _, grei := range ggi.GREInfos {
					fmt.Println("GRE ID       :", grei.ID)
					fmt.Println("IN GROUP     :", ggi.Name)
					fmt.Println("NAME         :", grei.Name)
					fmt.Println("ARGS         :", grei.Args)
					fmt.Println("REQUESTED BY :", grei.RequestedBy)
					fmt.Println("STATUS       :", greStatToStr(grei.Stat))
					fmt.Println("RESTARTED    :", grei.RestartedNum)
					startTime := ""
					if !grei.StartTime.IsZero() {
						startTime = fmt.Sprint(grei.StartTime)
					}
					fmt.Println("START AT     :", startTime)
					endTime := ""
					if greStatIsTerminated(grei.Stat) {
						endTime = fmt.Sprint(grei.EndTime)
					}
					fmt.Println("END AT       :", endTime)
					fmt.Printf("ERROR        : %v\n\n", grei.GREErr)
				}
			}
		} else { // ps
			fmt.Println("GRE ID        IN GROUP            NAME                START AT             STATUS")
			trimName := func(name string) string {
				if len(name) > 18 {
					name = name[:13] + "..."
				}
				return name
			}
			for _, ggi := range ggis {
				for _, grei := range ggi.GREInfos {

					created := grei.StartTime.Format("2006/01/02 15:04:05")
					stat := greStatToStr(grei.Stat)
					if grei.Stat == greStatExited {
						ret := ":OK"
						if len(grei.GREErr) != 0 {
							ret = ":ERR"
						}
						stat = stat + ret
					}
					d := grei.EndTime.Sub(grei.StartTime)
					stat = fmt.Sprintf("%-10s %v", stat, d)

					fmt.Printf("%s  %-18s  %-18s  %s  %s\n", grei.ID, trimName(ggi.Name), trimName(grei.Name), created, stat)
				}
			}
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPatternCmds() {
	for _, cmdStrs := range [][]string{
		{"stop", "[options] [GRE IDs ...|names ...]", "Stop one or more jobs on local/remote node"},
		{"rm", "[options] [GRE IDs ...|names ...]", "Remove one or more stopped jobs on local/remote node"},
		{"start", "[options] [GRE IDs ...|names ...]", "Start one or more stopped jobs on local/remote node"},
	} {
		cmdStrs := cmdStrs
		cmd := flag.NewFlagSet(newCmd(cmdStrs[0], cmdStrs[1], cmdStrs[2]), flag.ExitOnError)
		grgName := cmd.String("group", "*", "in which GRG, wildcard(*) is supported")

		action := func() error {
			lg := newLogger(log.DefaultStream, "main")
			conn := connectDaemon(providerID, lg)
			if conn == nil {
				return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
			}
			defer conn.Close()

			msg := cmdPatternAction{GRGName: *grgName, IDPattern: cmd.Args(), Cmd: cmdStrs[0]}
			var greids []*grgGREIDs
			if err := conn.SendRecv(&msg, &greids); err != nil {
				return err
			}

			var info string
			switch msg.Cmd {
			case "stop":
				info = "stopped"
			case "rm":
				info = "removed"
			case "start":
				info = "started"
			}
			var sb strings.Builder
			for _, ggi := range greids {
				str := strings.Join(ggi.GREIDs, "\n")
				if len(str) != 0 {
					fmt.Fprintln(&sb, str)
				}
			}
			if sb.Len() > 0 {
				fmt.Print(sb.String())
				fmt.Println(info)
			}
			return nil
		}
		cmds = append(cmds, subCmd{cmd, action})
	}
}

func addInfoCmd() {
	cmd := flag.NewFlagSet(newCmd("info", "", "Show gshell info on local/remote node"), flag.ExitOnError)

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		var info string
		if err := conn.SendRecv(cmdInfo{}, &info); err != nil {
			return err
		}
		fmt.Print(info)

		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addLogCmd() {
	cmd := flag.NewFlagSet(newCmd("log", "[options] <daemon|grg|GRE ID>", "Print target log on local/remote node"), flag.ExitOnError)
	follow := cmd.Bool("f", false, "follow and output appended data as the log grows")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no target provided, see gshell log --help")
		}
		target := args[0]
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(providerID, lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		msg := cmdLog{Target: target, Follow: *follow}
		if err := conn.Send(&msg); err != nil {
			return err
		}
		if *follow {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT)
			go func() {
				sig := <-sigChan
				lg.Debugf("signal: %s", sig.String())
				conn.Close()
			}()
			ioconn := as.NewStreamIO(conn)
			io.Copy(os.Stdout, ioconn)
			lg.Debugln("cmdLog: done")
		} else {
			var log []byte
			if err := conn.Recv(&log); err != nil {
				return err
			}
			fmt.Printf("%s", log)
		}

		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addMsgTraceCmd() {
	cmd := flag.NewFlagSet(newCmd("__mtrace", "<list>", "List traceable message types, including those in plugins"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 || args[0] != "list" {
			return errors.New("wrong usage, see gshell __mtrace --help")
		}
		if err := loadPlugins(pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "load plugins error: %v", err)
		}

		types := as.GetKnownMessageTypes()
		sort.Slice(types, func(i, j int) bool {
			return types[i] < types[j]
		})
		for i, name := range types {
			fmt.Println(i, name)
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPluginsCmd() {
	cmd := flag.NewFlagSet(newCmd("__plugin", "<list>", "List plugins"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 || args[0] != "list" {
			return errors.New("wrong usage, see gshell __plugin --help")
		}
		if err := loadPlugins(pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "load plugins error: %v", err)
		}

		plugins := strings.Join(listPlugins(), ",")
		if len(plugins) != 0 {
			fmt.Printf("%s", plugins)
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

// ShellMain is the main entry of gshell
func ShellMain() error {
	var traceList string
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.StringVar(&loglevel, "l", loglevel, "")
	flag.StringVar(&loglevel, "loglevel", loglevel, "")
	flag.StringVar(&providerID, "p", providerID, "")
	flag.StringVar(&providerID, "provider", providerID, "")
	flag.StringVar(&traceList, "trace", "", "")
	flag.StringVar(&pluginDir, "plugin", pluginDir, "")

	addIDCmd()
	addExecCmd()
	addDaemonCmd()
	addListCmd()
	addStartCmd()
	addRepoCmd()
	addRunCmd()
	addKillCmd()
	addPsCmd()
	addPatternCmds()
	addInfoCmd()
	addLogCmd()
	addJoblistCmd()
	addMsgTraceCmd()
	addPluginsCmd()

	usage := func() {
		const opt = `
gshell is a simple pure golang service framework for linux devices.

Running a gshell daemon on a board/VM/container makes it a node in the gshell service mesh.
Each node has an unique provider ID.

Each job runs in one dedicated GRE(Gshell Runtime Environment) which runs in a GRG(Gshell Runtime Group).
GREs can be grouped into one named GRG for better performance.

gshell enters interactive mode if no options and no commands provided.

Usage: [OPTIONS] COMMAND ...
OPTIONS:
  -v, --version
        Show version
  -l, --loglevel
        Loglevel, debug/info/warn/error (default "%s")
  -p, --provider
        Provider ID, run following command on the remote node with this ID (default "%s")
  --plugin
        Local path of plugins(.gp files) (default "%s")
`
		/* hide --trace option, which is for tracing gshell internal messages only
		--trace
			Comma seprated messages to be traced, use "gshell mtrace list" to show possbile values
		*/

		fmt.Printf(opt, loglevel, providerID, pluginDir)
		fmt.Println("COMMANDS:")
		for _, cmd := range cmds {
			name := cmd.Name()
			if !strings.HasPrefix(name, "__") {
				fmt.Println("  " + name)
			}
		}
	}
	flag.Usage = usage

	flag.Parse()
	args := flag.Args()
	if showVersion {
		fmt.Println(version)
		return nil
	}
	// no command, enter interactive mode
	if len(args) == 0 {
		if err := loadPlugins(pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "load plugins error: %v", err)
		}

		gsh, err := newShell()
		if err != nil {
			return err
		}

		if err := gsh.init(interp.Options{}); err != nil {
			return err
		}

		gsh.runREPL()
		return nil
	}

	var tokens []string
	if len(traceList) != 0 {
		fields := strings.Split(traceList, ",")
		for _, name := range fields {
			token, err := as.TraceMsgByName(name, 1)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Tracing <%s> with token %s\n\n", name, token)
			tokens = append(tokens, token)
		}
	}
	str := args[0]
	for _, cmd := range cmds {
		if str == strings.Split(cmd.Name(), " ")[0] {
			cmd.SetOutput(os.Stdout)
			cmd.Parse(args[1:])
			err := cmd.action()
			for _, token := range tokens {
				fmt.Printf("\nTraced records with token: %s\n", token)
				msgs, err := as.ReadTracedMsg(token)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println(msgs)
			}
			return err
		}
	}
	return fmt.Errorf("unknown command: %s, see gshell --help", str)
}
