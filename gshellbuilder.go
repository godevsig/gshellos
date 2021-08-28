package gshellos

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/d5/tengo/v2/stdlib"
	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/gshellos/lined"
	"github.com/godevsig/gshellos/log"
)

var (
	version           string
	workDir           = "/var/tmp/gshell/"
	logDir            = workDir + "logs/"
	debugService      func(lg *log.Logger)
	godevsigPublisher = "godevsig.org"
)

func init() {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(err)
	}
}

type shell struct {
	modules     *tengo.ModuleMap
	symbolTable *tengo.SymbolTable
	globals     []tengo.Object
}

func (sh *shell) initModules() {
	sh.modules = stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	sh.modules.AddMap(GetModuleMap(AllModuleNames()...))
}

func (sh *shell) addFunction(name string, fn tengo.CallableFunc) {
	symbol := sh.symbolTable.Define(name)
	sh.globals[symbol.Index] = &tengo.UserFunction{
		Name:  name,
		Value: fn,
	}
}

func (sh *shell) preCompile() {
	sh.symbolTable = tengo.NewSymbolTable()
	sh.globals = make([]tengo.Object, tengo.GlobalsSize)

	for _, v := range globalFuncs {
		sh.addFunction(v.name, v.fn)
	}
}

func (sh *shell) runREPL() {
	fileSet := parser.NewFileSet()
	var constants []tengo.Object

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
			srcFile := fileSet.AddFile("repl", -1, len(line))
			p := parser.NewParser(srcFile, []byte(line), nil)
			file, err := p.ParseFile()
			if err != nil {
				fmt.Println(err)
				continue
			}

			c := tengo.NewCompiler(srcFile, sh.symbolTable, constants, sh.modules, nil)
			if err := c.Compile(file); err != nil {
				fmt.Println(err)
				continue
			}

			bytecode := c.Bytecode()
			bytecode.RemoveDuplicates()
			machine := tengo.NewVM(bytecode, sh.globals, -1)
			if err := machine.Run(); err != nil {
				fmt.Println(err)
				continue
			}
			constants = bytecode.Constants
		}
	}
}

func (sh *shell) compile(inputFile string) (bytecode *tengo.Bytecode, err error) {
	src, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return
	}
	if len(src) > 1 && string(src[:2]) == "#!" {
		copy(src, "//")
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(inputFile), -1, len(src))

	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return
	}

	c := tengo.NewCompiler(srcFile, sh.symbolTable, nil, sh.modules, nil)
	c.EnableFileImport(true)
	c.SetImportDir(filepath.Dir(inputFile))

	if err = c.Compile(file); err != nil {
		return
	}

	bytecode = c.Bytecode()
	bytecode.RemoveDuplicates()
	return
}

func (sh *shell) compileAndSave(inputFile, outputFile string) (err error) {
	bytecode, err := sh.compile(inputFile)
	if err != nil {
		return
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return
	}
	defer out.Close()

	err = bytecode.Encode(out)
	if err != nil {
		return
	}
	fmt.Println(outputFile)
	return
}

func (sh *shell) compileAndExec(inputFile string) (err error) {
	bytecode, err := sh.compile(inputFile)
	if err != nil {
		return
	}

	machine := tengo.NewVM(bytecode, sh.globals, -1)
	err = machine.Run()
	return
}

func (sh *shell) execCompiled(inputFile string) (err error) {
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return
	}

	bytecode := &tengo.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(data), sh.modules)
	if err != nil {
		return
	}

	machine := tengo.NewVM(bytecode, sh.globals, -1)
	err = machine.Run()
	return
}

func newShell() *shell {
	sh := &shell{}
	sh.initModules()
	sh.preCompile()
	return sh
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
var loglevel = flag.String("loglevel", "error", "debug/info/warn/error")

func newLogger(logStream *log.Stream, loggerName string) *log.Logger {
	level := log.Linfo
	switch *loglevel {
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
	cmd := flag.NewFlagSet(newCmd("daemon", "[options]", "start gshell daemon"), flag.ExitOnError)
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
		logStream.SetOutput("file:" + workDir + "daemon.log")
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

func addCompileCmd() {
	cmd := flag.NewFlagSet(newCmd("compile", "<file.gsh>", "compile <file.gsh> to byte code <file>"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}
		file := args[0]
		inputFile, _ := filepath.Abs(file)
		if filepath.Ext(inputFile) != ".gsh" {
			return errors.New("wrong file suffix, see --help")
		}
		sh := newShell()
		outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
		return sh.compileAndSave(inputFile, outputFile)
	}
	cmds = append(cmds, subCmd{cmd, action})
}

func addExecCmd() {
	cmd := flag.NewFlagSet(newCmd("exec", "<file[.gsh]> [args...]", "run <file[.gsh]> in a local standalone VM"), flag.ExitOnError)

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}

		file := args[0]
		inputFile, _ := filepath.Abs(file)
		os.Args = args[1:] // pass os.Args down
		sh := newShell()
		if filepath.Ext(inputFile) == ".gsh" {
			return sh.compileAndExec(inputFile)
		}
		return sh.execCompiled(inputFile)
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
		logStream.SetOutput("file:" + workDir + "gre.log")
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

func addRunCmd() {
	cmd := flag.NewFlagSet(newCmd("run", "[options] <file[.gsh]> [args...]", "run <file[.gsh]> in a new VM in gre"), flag.ExitOnError)
	providerID := cmd.String("p", "self", "specify the system to run the command by provider ID")
	greName := cmd.String("e", "master", "create new or use existing gre(gshell runtime environment) in which the VM runs")
	interactive := cmd.Bool("i", false, "enter interactive mode")
	autoRemove := cmd.Bool("rm", false, "automatically remove the VM when it exits")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}

		file := args[0]
		inputFile, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		sh := newShell()
		var b bytes.Buffer
		if filepath.Ext(inputFile) == ".gsh" {
			bytecode, err := sh.compile(inputFile)
			if err != nil {
				return err
			}
			err = bytecode.Encode(&b)
			if err != nil {
				return err
			}
		} else {
			f, err := os.Open(inputFile)
			if err != nil {
				return err
			}
			defer f.Close()
			b.ReadFrom(f)
		}
		byteCode := b.Bytes()
		cmd := cmdRun{
			greCmdRun: greCmdRun{
				File:        inputFile,
				Args:        args[1:],
				Interactive: *interactive,
				AutoRemove:  *autoRemove,
				ByteCode:    byteCode,
			},
			GreName: *greName,
		}

		lg := newLogger(log.DefaultStream, "main")
		opts := []as.Option{
			as.WithLogger(lg),
		}
		c := as.NewClient(opts...)
		var conn as.Connection
		if *providerID == "self" { // local
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
		} else { // remote
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon", *providerID)
		}
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
	cmd := flag.NewFlagSet(newCmd("ps", "[options] [VMIDs ...|names ...]", "show VM instances by VM ID or name"), flag.ExitOnError)
	providerID := cmd.String("p", "self", "specify the system to run the command by provider ID")
	greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

	action := func() error {
		lg := newLogger(log.DefaultStream, "main")
		opts := []as.Option{
			as.WithLogger(lg),
		}
		c := as.NewClient(opts...)
		var conn as.Connection
		if *providerID == "self" { // local
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
		} else { // remote
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon", *providerID)
		}
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
		{"kill", "[options] [VMIDs ...|names ...]", "abort the execution of one or more VMs"},
		{"rm", "[options] [VMIDs ...|names ...]", "remove one or more stopped VMs, running VM can not be removed"},
		{"restart", "[options] [VMIDs ...|names ...]", "restart one or more stopped VMs, no effect on a running VM"},
	} {
		cmdStrs := cmdStrs
		cmd := flag.NewFlagSet(newCmd(cmdStrs[0], cmdStrs[1], cmdStrs[2]), flag.ExitOnError)
		providerID := cmd.String("p", "self", "specify the system to run the command by provider ID")
		greName := cmd.String("e", "*", "in which gre(gshell runtime environment)")

		action := func() error {
			lg := newLogger(log.DefaultStream, "main")
			opts := []as.Option{
				as.WithLogger(lg),
			}
			c := as.NewClient(opts...)
			var conn as.Connection
			if *providerID == "self" { // local
				conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
			} else { // remote
				conn = <-c.Discover(godevsigPublisher, "gshellDaemon", *providerID)
			}
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

func addTailfCmd() {
	cmd := flag.NewFlagSet(newCmd("tailf", "[options] <daemon|gre|VMID>", "print logs of the daemon/gre or print outputs of the VM by VMID"), flag.ExitOnError)
	providerID := cmd.String("p", "self", "specify the system to run the command by provider ID")

	action := func() error {
		args := cmd.Args()
		if len(args) == 0 {
			return errors.New("no target provided, see --help")
		}
		target := args[0]
		lg := newLogger(log.DefaultStream, "main")
		opts := []as.Option{
			as.WithLogger(lg),
		}
		c := as.NewClient(opts...)
		var conn as.Connection
		if *providerID == "self" { // local
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon")
		} else { // remote
			conn = <-c.Discover(godevsigPublisher, "gshellDaemon", *providerID)
		}
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
		sh := newShell()
		sh.runREPL()
		return nil
	}

	addDeamonCmd()
	addListCmd()
	addIDCmd()
	addCompileCmd()
	addExecCmd()
	addStartCmd()
	addRunCmd()
	addPsCmd()
	addPatternCmds()
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
		help := `gshell is an interpreter in Golang style syntax and a supervision tool to
manage .gsh apps running in gshell VM(tengo based virtual machine).

Each .gsh file runs in a separate new VM so one VM's crash does not impact others.
VMs run in gshell runtime environment(gre), which is essentially a share memory
space in which the communications between VMs are fast.

gshell daemon command starts a daemon which is supposed to run at each system
so that gshell run command can run .gsh file in a remote system with specified
provider ID, this remote run also works for NAT network.

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
