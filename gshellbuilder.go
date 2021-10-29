package gshellos

import (
	"crypto/md5"
	_ "embed" // go embed
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/traefik/yaegi/interp"
)

//go:embed bin/rev
var commitRev string

//go:embed bin/gittag
var version string

//go:embed bin/buildtag
var buildTags string

var (
	workDir           = "/var/tmp/gshell"
	loglevel          = "error"
	providerID        = "self"
	debugService      func(lg *log.Logger)
	godevsigPublisher = "godevsig"
)

func init() {
	if len(commitRev) == 0 {
		commitRev = "devel"
	}
	if len(version) == 0 {
		version = commitRev[:5]
	}
}

func getSelfID(opts []as.Option) (selfID string, err error) {
	opts = append(opts, as.WithScope(as.ScopeProcess|as.ScopeOS))
	c := as.NewClient(opts...).SetDiscoverTimeout(0)
	conn := <-c.Discover(as.BuiltinPublisher, as.SrvProviderInfo)
	if conn == nil {
		err = as.ErrServiceNotFound(as.BuiltinPublisher, as.SrvProviderInfo)
		return
	}
	defer conn.Close()

	err = conn.SendRecv(&as.ReqProviderInfo{}, &selfID)
	return
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

func addDeamonCmd() {
	cmd := flag.NewFlagSet(newCmd("daemon", "[options]", "Start local gshell daemon"), flag.ExitOnError)
	rootRegistry := cmd.Bool("root", false, "enable root registry service")
	registryAddr := cmd.String("registry", "", "root registry address")
	lanBroadcastPort := cmd.String("bcast", "", "broadcast port for LAN")
	codeRepo := cmd.String("repo", "", "code repo https address in format site/org/proj/branch")
	updateURL := cmd.String("update", "", "url of artifacts to update gshell")

	action := func() error {
		cmdArgs := os.Args
		scope := as.ScopeAll
		if len(*registryAddr) == 0 {
			scope &= ^as.ScopeWAN // not ScopeWAN
		}
		if len(*lanBroadcastPort) == 0 {
			scope &= ^as.ScopeLAN // not ScopeLAN
		}
		if len(*codeRepo) != 0 || len(*updateURL) != 0 || *rootRegistry {
			if scope&as.ScopeWAN != as.ScopeWAN {
				return errors.New("root registry address not set")
			}
		}

		var repoInfo []string
		if len(*codeRepo) != 0 {
			repoInfo = strings.Split(*codeRepo, "/")
			if len(repoInfo) != 4 {
				return errors.New("wrong repo format")
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

		logStream := log.NewStream("daemon")
		if err := logStream.SetOutput("file:" + workDir + "/logs/daemon.log"); err != nil {
			return err
		}
		lg := newLogger(logStream, "daemon")
		lg.Infof("daemon version: %s", version)

		opts := []as.Option{
			as.WithScope(scope),
			as.WithLogger(lg),
		}
		if len(*registryAddr) != 0 {
			opts = append(opts, as.WithRegistryAddr(*registryAddr))
		}
		s := as.NewServer(opts...).
			SetPublisher(godevsigPublisher).
			EnableServiceLister()
		if len(*lanBroadcastPort) != 0 {
			s.SetBroadcastPort(*lanBroadcastPort)
		}
		if scope&as.ScopeWAN == as.ScopeWAN || scope&as.ScopeLAN == as.ScopeLAN {
			s.EnableAutoReverseProxy()
		}
		if *rootRegistry {
			s.EnableRootRegistry()
		}

		if len(repoInfo) == 4 {
			crs := &codeRepoSvc{repoInfo: repoInfo}
			if err := s.Publish("codeRepo",
				codeRepoKnownMsgs,
				as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(crs) }),
			); err != nil {
				return err
			}
		}

		if len(*updateURL) != 0 {
			updtr := &updater{urlFmt: *updateURL, lg: lg}
			if err := s.Publish("updater",
				updaterKnownMsgs,
				as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(updtr) }),
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
				err := conn.SendRecv(tryUpdate{revInuse: commitRev, arch: runtime.GOARCH}, &gshellbin)
				conn.Close()
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

				lg.Debugf(RunShCmd("ls -lh " + newFile))
				lg.Infof("updating gshell version...")

				if err := os.Rename(newFile, exe); err != nil {
					lg.Infof("failed to rename new gshell to %s: %s", exe, err)
					output := RunShCmd("mv -f " + newFile + " " + exe)
					if len(output) != 0 {
						lg.Warnf("failed to mv new gshell to %s: %s", exe, output)
						continue
					}
				}
				if output := RunShCmd("chmod ugo+s " + exe); len(output) != 0 {
					lg.Warnf("failed to set gshell bin permission : %s", output)
					continue
				}

				updateChan = make(chan struct{})
				s.Close()
				cmd := cmdArgs[0]
				args := cmdArgs[1:]
				if cmdArgs[0] == "gshell.tester" {
					args = append([]string{"-test.run", "^TestRunMain$", "--"}, args...)
				}
				if err := exec.Command(cmd, args...).Start(); err != nil {
					lg.Errorf("start new gshell failed: %v", err)
				} else {
					lg.Infof("new version gshell started")
				}
				close(updateChan)
				return
			}
		}()

		gd := &daemon{
			lg: lg,
		}
		if err := s.Publish("gshellDaemon",
			daemonKnownMsgs,
			as.OnNewStreamFunc(gd.onNewStream),
		); err != nil {
			return err
		}
		if debugService != nil {
			go debugService(lg)
		}

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
		opts := []as.Option{
			as.WithScope(as.ScopeProcess | as.ScopeOS),
			as.WithLogger(newLogger(log.DefaultStream, "main")),
		}
		selfID, _ := getSelfID(opts)

		c := as.NewClient(opts...)
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
				ss := strings.Split(svc, "_")
				fmt.Printf("%-24s  %-24s  %-12s  %4b\n", trimName(ss[0], 24), trimName(ss[1], 24), ss[2], *p)
			}
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addIDCmd() {
	cmd := flag.NewFlagSet(newCmd("id", "", "Print self provider ID"), flag.ExitOnError)

	action := func() error {
		opts := []as.Option{
			as.WithLogger(newLogger(log.DefaultStream, "main")),
		}
		selfID, err := getSelfID(opts)
		if err != nil {
			selfID = "NA"
		}
		fmt.Println(selfID)
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addExecCmd() {
	cmd := flag.NewFlagSet(newCmd("exec", "<file.go> [args...]", "Run <file.go> in a local VM"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}

		file := args[0]
		sh := newShell(interp.Options{Args: args})
		err := sh.runFile(file)
		if p, ok := err.(interp.Panic); ok {
			err = fmt.Errorf("%w\n%s", err, string(p.Stack))
		}
		return err
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addStartCmd() {
	cmd := flag.NewFlagSet(newCmd("__start", "[options]", "Start named gre"), flag.ExitOnError)
	greName := cmd.String("e", "", "create new gre(gshell runtime environment)")

	action := func() error {
		if len(*greName) == 0 {
			return errors.New("no gre name, see --help")
		}

		logStream := log.NewStream("gre")
		logStream.SetOutput("file:" + workDir + "/logs/gre.log")
		lg := newLogger(logStream, "gre-"+*greName)
		opts := []as.Option{
			as.WithScope(as.ScopeOS),
			as.WithLogger(lg),
		}

		s := as.NewServer(opts...).SetPublisher(godevsigPublisher)
		gre := &gre{
			name: *greName,
			lg:   lg,
			vms:  make(map[string]*vmCtl),
		}

		if err := s.Publish("gre-"+*greName,
			greKnownMsgs,
			as.OnNewStreamFunc(gre.onNewStream),
		); err != nil {
			return err
		}
		if debugService != nil {
			go debugService(lg)
		}
		return s.Serve()
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func connectDaemon(lg *log.Logger) (conn as.Connection) {
	c := as.NewClient(as.WithLogger(lg)).SetDiscoverTimeout(0)
	if providerID == "self" { // local
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
	} else { // remote
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon", providerID)
	}
	return
}

func getCodeRepoAddr(lg as.Logger) (addr string, err error) {
	addr = "NA"
	c := as.NewClient(as.WithLogger(lg)).SetDiscoverTimeout(0)
	conn := <-c.Discover(godevsigPublisher, "codeRepo")
	if conn == nil {
		err = as.ErrServiceNotFound(godevsigPublisher, "codeRepo")
		return
	}
	defer conn.Close()

	err = conn.SendRecv(codeRepoAddr{}, &addr)
	return
}

func addRepoCmd() {
	cmd := flag.NewFlagSet(newCmd("repo", "", "Print central code repo https address"), flag.ExitOnError)

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		addr, _ := getCodeRepoAddr(lg)
		fmt.Println(addr)
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addKillCmd() {
	cmd := flag.NewFlagSet(newCmd("kill", "[options] names ...", "Terminate the named gre(s) on local/remote system"), flag.ExitOnError)
	force := cmd.Bool("f", false, "force terminate even if there are still running VMs")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no gre specified, see --help")
		}

		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		cmd := cmdKill{
			GreNames: args,
			Force:    *force,
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

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func addRunCmd() {
	cmd := flag.NewFlagSet(newCmd("run",
		"[options] <file.go> [args...]",
		"Look for file.go in local file system or else in `gshell repo`,",
		"run it in a new VM in specified gre on local/remote system"),
		flag.ExitOnError)
	greName := cmd.String("e", "<random>", "create new or use existing gre(gshell runtime environment)")
	rtPriority := cmd.String("rt", "", `Set the gre to SCHED_RR min/max priority 1/99 on new gre creation
Caution: gshell daemon must be started as root to set realtime attributes`)
	interactive := cmd.Bool("i", false, "enter interactive mode")
	autoRemove := cmd.Bool("rm", false, "automatically remove the VM when it exits")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}
		gre := *greName
		if strings.Contains(gre, "*") {
			return errors.New("wrong use of wildcard(*), see --help")
		}
		if len(gre) == 0 || gre == "<random>" {
			gre = randStringRunes(6)
		}
		if len(*rtPriority) != 0 {
			pri, err := strconv.Atoi(*rtPriority)
			if err != nil || pri < 1 || pri > 99 {
				return errors.New("wrong SCHED_RR priority, see man chrt")
			}
		}

		lg := newLogger(log.DefaultStream, "main")
		file := args[0]
		var byteCode []byte
		var err error
		if !strings.HasSuffix(file, ".go") {
			return errors.New("wrong file suffix, see --help")
		}

		byteCode, err = os.ReadFile(file)
		if err != nil {
			c := as.NewClient(as.WithLogger(lg)).SetDiscoverTimeout(0)
			conn := <-c.Discover(godevsigPublisher, "codeRepo")
			if conn == nil {
				return errors.New("file not found")
			}
			defer conn.Close()

			if err := conn.SendRecv(getFileContent{file}, &byteCode); err != nil {
				return err
			}
		}

		cmd := cmdRun{
			greCmdRun: greCmdRun{
				File:        file,
				Args:        args,
				Interactive: *interactive,
				AutoRemove:  *autoRemove,
				ByteCode:    rmShebang(byteCode),
			},
			GreName:    gre,
			RtPriority: *rtPriority,
		}

		conn := connectDaemon(lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		if err := conn.Send(&cmd); err != nil {
			return err
		}

		if !*interactive {
			var vmid string
			if err := conn.Recv(&vmid); err != nil {
				return err
			}
			fmt.Println(vmid)
			return nil
		}

		ioconn := as.NewStreamIO(conn)
		lg.Debugln("enter interactive io")
		go io.Copy(ioconn, os.Stdin)
		io.Copy(os.Stdout, ioconn)
		lg.Debugln("exit interactive io")
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPsCmd() {
	cmd := flag.NewFlagSet(newCmd("ps", "[options] [VMIDs ...|names ...]", "Show VM instances by VM ID or name on local/remote system"), flag.ExitOnError)
	greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(lg)
		if conn == nil {
			return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
		}
		defer conn.Close()

		msg := cmdQuery{GreName: *greName, IDPattern: cmd.Args()}
		var gvis []*greVMInfo
		if err := conn.SendRecv(&msg, &gvis); err != nil {
			return err
		}

		if len(msg.IDPattern) != 0 { // info
			for _, gvi := range gvis {
				for _, vmi := range gvi.VMInfos {
					fmt.Println("ID        :", vmi.ID)
					fmt.Println("IN GRE    :", gvi.Name)
					fmt.Println("NAME      :", vmi.Name)
					fmt.Println("ARGS      :", vmi.Args)
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
			fmt.Println("VM ID         IN GRE            NAME              START AT             STATUS")
			trimName := func(name string) string {
				if len(name) > 16 {
					name = name[:13] + "..."
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

					fmt.Printf("%s  %-16s  %-16s  %s  %s\n", vmi.ID, trimName(gvi.Name), trimName(vmi.Name), created, stat)
				}
			}
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPatternCmds() {
	for _, cmdStrs := range [][]string{
		{"stop", "[options] [VMIDs ...|names ...]", "Call `func Stop()` to stop one or more VMs on local/remote system"},
		{"rm", "[options] [VMIDs ...|names ...]", "Remove one or more stopped VMs on local/remote system"},
		{"restart", "[options] [VMIDs ...|names ...]", "Restart one or more stopped VMs on local/remote system"},
	} {
		cmdStrs := cmdStrs
		cmd := flag.NewFlagSet(newCmd(cmdStrs[0], cmdStrs[1], cmdStrs[2]), flag.ExitOnError)
		greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

		action := func() error {
			lg := newLogger(log.DefaultStream, "main")
			conn := connectDaemon(lg)
			if conn == nil {
				return as.ErrServiceNotFound(godevsigPublisher, "gshellDaemon")
			}
			defer conn.Close()

			msg := cmdPatternAction{GreName: *greName, IDPattern: cmd.Args(), Cmd: cmdStrs[0]}
			var vmids []*greVMIDs
			if err := conn.SendRecv(&msg, &vmids); err != nil {
				return err
			}

			var info string
			switch msg.Cmd {
			case "stop":
				info = "stopped"
			case "rm":
				info = "removed"
			case "restart":
				info = "restarted"
			}
			var sb strings.Builder
			for _, gvi := range vmids {
				str := strings.Join(gvi.VMIDs, "\n")
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
	cmd := flag.NewFlagSet(newCmd("info", "", "Show gshell info on local/remote system"), flag.ExitOnError)

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(lg)
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
	cmd := flag.NewFlagSet(newCmd("log", "[options] <daemon|gre|VMID>", "Print target log on local/remote system"), flag.ExitOnError)
	follow := cmd.Bool("f", false, "follow and output appended data as the log grows")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no target provided, see --help")
		}
		target := args[0]
		lg := newLogger(log.DefaultStream, "main")
		conn := connectDaemon(lg)
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

// ShellMain is the main entry of gshell
func ShellMain() error {
	// no arg, shell mode
	if len(os.Args) == 1 {
		newShell(interp.Options{}).runREPL()
		return nil
	}

	flag.StringVar(&workDir, "wd", workDir, "set working directory")
	flag.StringVar(&loglevel, "loglevel", loglevel, "debug/info/warn/error")
	flag.StringVar(&providerID, "p", providerID, "provider ID to specify a remote system")

	addDeamonCmd()
	addListCmd()
	addIDCmd()
	addExecCmd()
	addStartCmd()
	addRepoCmd()
	addKillCmd()
	addRunCmd()
	addPsCmd()
	addPatternCmds()
	addInfoCmd()
	addLogCmd()

	usage := func() {
		fmt.Println("COMMANDS:")
		for _, cmd := range cmds {
			name := cmd.Name()
			if !strings.HasPrefix(name, "__") {
				fmt.Println("  " + name)
			}
		}
	}

	switch os.Args[1] {
	case "-h", "--help":
		help := `gshell is a supervision tool to run .go apps by a VM(yaegi interpreter) in
gshell runtime environment(gre), which is essentially a shared memory space.

gshell daemon starts a daemon which is supposed to run on each system so that
gshell run/ps/... commands can run on a remote system with specified provider ID.

gshell enters interactive mode if no options and no commands provided.
`
		fmt.Println(help)
		fmt.Println("Usage: [OPTIONS] COMMAND ...")
		fmt.Println("OPTIONS:")
		flag.PrintDefaults()
		usage()
		return nil
	case "-v", "--version":
		fmt.Println(version)
		return nil
	default:
		flag.Parse()
		if err := os.MkdirAll(workDir+"/logs", 0755); err != nil {
			panic(err)
		}
		args := flag.Args()
		if len(args) == 0 {
			return errors.New("no command provided, see --help")
		}
		str := args[0]
		for _, cmd := range cmds {
			if str == strings.Split(cmd.Name(), " ")[0] {
				cmd.SetOutput(os.Stdout)
				cmd.Parse(args[1:])
				if err := cmd.action(); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("unknown command: %s, see --help", str)
	}
}
