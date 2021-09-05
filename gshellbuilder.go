package gshellos

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/lined"
	"github.com/godevsig/gshellos/log"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var (
	version           string
	buildTags         string
	workDir           = "/var/tmp/gshell"
	loglevel          = "error"
	providerID        = "self"
	debugService      func(lg *log.Logger)
	godevsigPublisher = "godevsig.org"
)

type shell struct {
	*interp.Interpreter
}

func newShell(opt interp.Options) *shell {
	os.Setenv("YAEGI_SPECIAL_STDIO", "1")
	i := interp.New(opt)
	if err := i.Use(stdlib.Symbols); err != nil {
		panic(err)
	}
	if err := i.Use(extension.Symbols); err != nil {
		panic(err)
	}
	i.ImportUsed()
	sh := &shell{Interpreter: i}
	return sh
}

func (sh *shell) runREPL() {
	ctx, cancel := context.WithCancel(context.Background())
	end := make(chan struct{}) // channel to terminate the REPL
	defer close(end)
	sig := make(chan os.Signal, 1) // channel to trap interrupt signal (Ctrl-C)

	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	go func() {
		for {
			select {
			case <-sig:
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
			case <-end:
				return
			}
		}
	}()

	led := lined.NewEditor(lined.Cfg{
		Prompt: ">> ",
	})
	defer led.Close()

	for {
		line, err := led.Readline()
		if errors.Is(err, io.EOF) {
			break
		}
		if len(line) != 0 {
			_, err := sh.EvalWithContext(ctx, line)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func rmShebang(b []byte) []byte {
	if len(b) >= 2 {
		if string(b[:2]) == "#!" {
			copy(b, "//")
		}
	}
	return b
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func (sh *shell) runFile(path string) error {
	if isFile(path) {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = sh.Eval(string(rmShebang(b)))
		return err
	}

	_, err := sh.EvalPath(path)
	return err
}

func getSelfID(opts []as.Option) (selfID string, err error) {
	opts = append(opts, as.WithScope(as.ScopeProcess|as.ScopeOS))
	c := as.NewClient(opts...).SetDiscoverTimeout(0)
	conn := <-c.Discover(as.BuiltinPublisher, "providerInfo")
	if conn == nil {
		err = as.ErrServiceNotFound
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

func newCmd(name, usage, help string) string {
	head := fmt.Sprintf("%s %s", name, usage)
	return fmt.Sprintf("%s\n\t%s", head, help)
}

func addDeamonCmd() {
	cmd := flag.NewFlagSet(newCmd("daemon", "[options]", "start local gshell daemon"), flag.ExitOnError)
	rootRegistry := cmd.Bool("root", false, "enable root registry service")
	registryAddr := cmd.String("registry", "", "root registry address")
	lanBroadcastPort := cmd.String("bcast", "", "broadcast port for LAN")

	action := func() error {
		if len(*registryAddr) == 0 {
			return errors.New("root registry address not set")
		}
		if len(*lanBroadcastPort) == 0 {
			return errors.New("lan broadcast port not set")
		}

		logStream := log.NewStream("daemon")
		logStream.SetOutput("file:" + workDir + "/logs/daemon.log")
		lg := newLogger(logStream, "daemon")

		opts := []as.Option{
			as.WithRegistryAddr(*registryAddr),
			as.WithLogger(lg),
		}
		s := as.NewServer(opts...).
			SetPublisher(godevsigPublisher).
			SetBroadcastPort(*lanBroadcastPort).
			EnableAutoReverseProxy().
			EnableServiceLister()
		if *rootRegistry {
			s.EnableRootRegistry()
		}

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

		return s.Serve()
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addListCmd() {
	cmd := flag.NewFlagSet(newCmd("list", "[options]", "list services in all scopes"), flag.ExitOnError)
	verbose := cmd.Bool("v", false, "show verbose info")
	publisher := cmd.String("p", "*", "publisher name, can be wildcard")
	service := cmd.String("s", "*", "service name, can be wildcard")

	action := func() error {
		opts := []as.Option{
			as.WithLogger(newLogger(log.DefaultStream, "main")),
		}
		selfID, err := getSelfID(opts)
		if err != nil {
			return err
		}

		c := as.NewClient(opts...)
		conn := <-c.Discover(as.BuiltinPublisher, "serviceLister")
		if conn == nil {
			return as.ErrServiceNotFound
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
			fmt.Println("PUBLISHER           SERVICE             PROVIDER      WLOP(SCOPE)")
			for _, svc := range names {
				p := list[svc]
				if p == nil {
					panic("nil p")
				}
				ss := strings.Split(svc, "_")
				fmt.Printf("%-18s  %-18s  %-12s  %4b\n", trimName(ss[0], 18), trimName(ss[1], 18), ss[2], *p)
			}
		}
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addIDCmd() {
	cmd := flag.NewFlagSet(newCmd("id", "", "print self provider ID"), flag.ExitOnError)

	action := func() error {
		opts := []as.Option{
			as.WithLogger(newLogger(log.DefaultStream, "main")),
		}
		selfID, err := getSelfID(opts)
		if err != nil {
			return err
		}
		fmt.Println(selfID)
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addExecCmd() {
	cmd := flag.NewFlagSet(newCmd("exec", "<file.go> [args...]", "run <file.go> in a local VM"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}

		file := args[0]
		os.Args = args // pass os.Args down
		sh := newShell(interp.Options{})
		return sh.runFile(file)
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addStartCmd() {
	cmd := flag.NewFlagSet(newCmd("__start", "[options]", "start named gre"), flag.ExitOnError)
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

func getConn(lg *log.Logger) (conn as.Connection) {
	c := as.NewClient(as.WithLogger(lg)).SetDiscoverTimeout(0)
	if providerID == "self" { // local
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
	} else { // remote
		conn = <-c.Discover(godevsigPublisher, "gshellDaemon", providerID)
	}
	return
}

func addRunCmd() {
	cmd := flag.NewFlagSet(newCmd("run", "[options] <file.go> [args...]", "run <file.go> in a new VM in specified gre on local/remote system"), flag.ExitOnError)
	greName := cmd.String("e", "master", "create new or use existing gre(gshell runtime environment)")
	interactive := cmd.Bool("i", false, "enter interactive mode")
	autoRemove := cmd.Bool("rm", false, "automatically remove the VM when it exits")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}

		file := args[0]
		byteCode, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		cmd := cmdRun{
			greCmdRun: greCmdRun{
				File:        file,
				Args:        args,
				Interactive: *interactive,
				AutoRemove:  *autoRemove,
				ByteCode:    rmShebang(byteCode),
			},
			GreName: *greName,
		}

		lg := newLogger(log.DefaultStream, "main")
		conn := getConn(lg)
		if conn == nil {
			return as.ErrServiceNotFound
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

		lg.Debugln("enter interactive io")
		go io.Copy(conn, os.Stdin)
		io.Copy(os.Stdout, conn)
		lg.Debugln("exit interactive io")
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPsCmd() {
	cmd := flag.NewFlagSet(newCmd("ps", "[options] [VMIDs ...|names ...]", "show VM instances by VM ID or name on local/remote system"), flag.ExitOnError)
	greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := getConn(lg)
		if conn == nil {
			return as.ErrServiceNotFound
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
		return nil
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addPatternCmds() {
	for _, cmdStrs := range [][]string{
		{"kill", "[options] [VMIDs ...|names ...]", "abort the execution of one or more VMs on local/remote system"},
		{"rm", "[options] [VMIDs ...|names ...]", "remove one or more stopped VMs on local/remote system"},
		{"restart", "[options] [VMIDs ...|names ...]", "restart one or more stopped VMs on local/remote system"},
	} {
		cmdStrs := cmdStrs
		cmd := flag.NewFlagSet(newCmd(cmdStrs[0], cmdStrs[1], cmdStrs[2]), flag.ExitOnError)
		greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

		action := func() error {
			lg := newLogger(log.DefaultStream, "main")
			conn := getConn(lg)
			if conn == nil {
				return as.ErrServiceNotFound
			}
			defer conn.Close()

			msg := cmdPatternAction{GreName: *greName, IDPattern: cmd.Args(), Cmd: cmdStrs[0]}
			var vmids []*greVMIDs
			if err := conn.SendRecv(&msg, &vmids); err != nil {
				return err
			}

			var info string
			switch msg.Cmd {
			case "kill":
				info = "killed"
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
	cmd := flag.NewFlagSet(newCmd("info", "", "show gshell info on local/remote system"), flag.ExitOnError)

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		conn := getConn(lg)
		if conn == nil {
			return as.ErrServiceNotFound
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

func addTailfCmd() {
	cmd := flag.NewFlagSet(newCmd("tailf", "[options] <daemon|gre|VMID>", "print logs on local/remote system"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no target provided, see --help")
		}
		target := args[0]
		lg := newLogger(log.DefaultStream, "main")
		conn := getConn(lg)
		if conn == nil {
			return as.ErrServiceNotFound
		}
		defer conn.Close()

		msg := cmdTailf{Target: target}
		if err := conn.Send(&msg); err != nil {
			return err
		}
		io.Copy(os.Stdout, conn)
		lg.Debugln("cmdTailf: done")

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
	addRunCmd()
	addPsCmd()
	addPatternCmds()
	addInfoCmd()
	addTailfCmd()

	if len(version) == 0 {
		version = "development"
	}

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
gshell run/ps/kill... commands can run on a remote system with specified provider ID.

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
