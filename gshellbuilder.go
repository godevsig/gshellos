package gshellos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/godevsig/gshellos/lined"
	"github.com/godevsig/gshellos/log"
	sm "github.com/godevsig/gshellos/scalamsg"
)

var version string
var debugService func(lg *log.Logger)

const (
	usage = `gshell is an interpreter in Golang style syntax and a supervision
tool to manage .gsh apps running in gshell VM(tengo based virtual machine).

SYNOPSIS
    gshell [OPTIONS] [COMMANDS]

Options:
    -c, --connect <hostname:port>
            Connect to the remote gre server instead of local gre server.
    -e, --gre <name>
            Specify the name of gre(gshell runtime environment) instance.
    -u, --upstream <hostname:port>
            Set the upstream gshell server address.
    -d, --debug     Enable debug loglevel, -d -d makes more verbose log.
    -h, --help      Show this message.
    -v, --version   Show version information.

Server and gre management commands:
    server [port]
            Start gre server for local connection, also accept remote
            connection if [port] is provided.
    gre stop [-f]
            Terminate the named gre(by -e) instacne.
            -f    force terminate even if there are still running VMs
    gre save <file>
            Save all .gsh apps in the named gre(by -e) to <file>.
    gre load <file>
            Load .gsh apps from <file> to the named gre(by -e) and run them.
    gre pull <[hostname:port://]name>
            Pull the [remote] gre <name> and combine it to the named gre(by -e).
    gre push <[hostname:port://]name>
            Push the named gre(by -e) and combine it to the [remote] gre <name>.
    gre suspend
            Suspend the execution of the named gre(by -e). All the running VMs
            in that gre will be suspended.
    gre resume
            Resume the execution of the named gre(by -e). All the running VMs
            in that gre will be resumed to run.
    gre priority <maximum|high|normal|low|minimum>
            Set the priority of the named gre(by -e). Default is normal.

The default "master" gre only supports save and load command.

Standalone commands:
    compile <file.gsh>
            Compile <file.gsh> into byte code <file>.
    exec <file[.gsh]> [args...]
            Run <file[.gsh]> in a local standalone VM.

Management commands of gre VM:
    run [-i --rm] <file[.gsh]> [args...]
            Run <file[.gsh]> in a new VM with its name set to base name of
            <file> in the designated gre and return a 12 hex digits VMID.
            -i    Enter interactive mode, keep STDIN and STDOUT
            open until <file[.gsh]> finishes execution.
            --rm  Automatically remove the VM when it exits.
            If no -e presents, the default "master" gre is used.
            The named gre instance will be created to run <file[.gsh]>
            if it has not been created by the remote/local gre server.
    ps [VMID1 VMID2 ...|name1 name2 ...]
            List VM instances in the local/remote gre server.
    kill <VMID1 VMID2 ...|name1 name2 ...>
            Abort the execution of one or more VMs.
    rm <VMID1 VMID2 ...|name1 name2 ...>
            Remove one or more stopped VMs and associated files, running
            VM can not be removed.
    restart <VMID1 VMID2 ...|name1 name2 ...>
            Restart one or more stopped VMs, no effect on a running VM.

Debugging commands:
    tailf <server|gre|VMID>
            Print logs of the server/gre or print outputs of the VM by VMID.

gshell enters interactive mode if no options and no commands provided.

Maintained by godevsig, see https://github.com/godevsig`
)

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

var savedOptions string

// ShellMain is the main entry of gshell
func ShellMain() error {
	args := os.Args

	// no arg, shell mode
	if len(args) == 1 {
		sh := newShell()
		sh.runREPL()
		return nil
	}
	args = args[1:] // shift

	remotegREServerAddr := ""
	greName := ""
	loglevel := log.Linfo
	logFlag := log.Ldefault
	if len(version) == 0 {
		version = "development"
	}

	// options
	for ; len(args) != 0; args = args[1:] {
		if args[0][0] != '-' {
			break // not option
		}
		switch args[0] {
		case "-h", "--help":
			fmt.Println(usage)
			return nil
		case "-v", "--version":
			fmt.Println(version)
			return nil
		case "-d", "--debug":
			savedOptions += "-d "
			loglevel--
			if loglevel <= log.Ltrace {
				loglevel = log.Ltrace
				logFlag = log.Lfileline
			}
		case "-c", "--connect": // -c, --connect <hostname:port>
			if len(args) > 1 {
				remotegREServerAddr = args[1]
				args = args[1:] // shift
			}
		case "-e", "--gre": // -e, --gre <name>
			if len(args) > 1 {
				greName = args[1]
				args = args[1:] // shift
			}
		default:
			return fmt.Errorf("unknown option %s, see --help", args[0])
		}
	}

	if loglevel != log.Linfo {
		gsStream.SetLoglevel("*", loglevel)
		gsStream.SetFlag(logFlag)
		gcStream.SetLoglevel("*", loglevel)
		gcStream.SetFlag(logFlag)
		greStream.SetLoglevel("*", loglevel)
		greStream.SetFlag(logFlag)
	}

	cmd := args[0]
	args = args[1:] // shift

	if cmd == "server" { // server [port]
		port := ""
		if len(args) > 0 {
			port = args[0]
		}
		if debugService != nil {
			go debugService(gsLogger)
		}
		return runServer(version, port)
	}

	if cmd == "compile" { // compile <file.gsh>
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

	if cmd == "exec" { // exec <file[.gsh]> [args...]
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}
		file := args[0]
		inputFile, _ := filepath.Abs(file)
		os.Args = args // pass os.Args down
		sh := newShell()
		if filepath.Ext(inputFile) == ".gsh" {
			return sh.compileAndExec(inputFile)
		}
		return sh.execCompiled(inputFile)
	}

	network := "unix"
	address := workDir + "gshelld.sock"
	if len(remotegREServerAddr) != 0 {
		network = "tcp"
		address = remotegREServerAddr
	}

	clientRun := func(client sm.Processor) error {
		return sm.DialRun(client, network, address,
			sm.ErrorAsEOF(),
			sm.WithLogger(gcLogger))
	}

	if cmd == "gre" {
		if len(args) == 0 {
			return errors.New("need gre sub-command, see --help")
		}
		if len(greName) == 0 {
			return errors.New("no gre name provided, need -e option")
		}

		subcmd := args[0]
		if greName == "master" {
			if subcmd != "save" && subcmd != "load" {
				return errors.New("master gre only supports save and load")
			}
		}
		args = args[1:] // shift
		var paras string
		switch subcmd {
		case "__start":
			if greName != "master" {
				if debugService != nil {
					go debugService(greLogger)
				}
				return rungre(greName)
			}
			return nil
		case "stop":
			if len(args) != 0 && args[0] == "-f" {
				paras = "-f"
			}
		case "save", "load":
			if len(args) == 0 {
				return errors.New("no file provided, see --help")
			}
			paras = args[0]
		case "suspend", "resume":
		case "priority":
			if len(args) == 0 {
				return errors.New("no priority provided, see --help")
			}
			switch args[0] {
			case "maximum", "high", "normal", "low", "minimum":
				paras = args[0]
			default:
				return errors.New("wrong priority, see --help")
			}
		default:
			return errors.New("unknown subcmd: " + subcmd)
		}

		greCmd := &greCmd{greName, subcmd, paras}
		return clientRun(greCmd)
	}

	if cmd == "run" { // run [-i] <file[.gsh]> [args...]
		interactive := false
		autoRemove := false
		for len(args) != 0 && args[0][0] == '-' {
			switch args[0] {
			case "-i":
				interactive = true
			case "--rm":
				autoRemove = true
			default:
				return fmt.Errorf("unknown option %s, see --help", args[0])
			}
			args = args[1:] // shift
		}
		if len(args) == 0 {
			return errors.New("no file provided, see --help")
		}
		file := args[0]
		inputFile, _ := filepath.Abs(file)
		if len(greName) == 0 {
			greName = "master"
		}
		cmdRun := &cmdRun{greName, inputFile, args, interactive, autoRemove, nil}
		return sm.DialRun(cmdRun, network, address,
			sm.ErrorAsEOF(),
			sm.RawMode(),
			sm.WithLogger(gcLogger))
	}

	if cmd == "ps" {
		cmdPs := cmdQuery{GreName: greName, IDPattern: args}
		return clientRun(cmdPs)
	}

	if cmd == "kill" || cmd == "rm" || cmd == "restart" {
		if len(args) == 0 {
			return errors.New("no vm id provided, see --help")
		}
		cmdAction := cmdPatternAction{GreName: greName, IDPattern: args, Cmd: cmd}
		return clientRun(cmdAction)
	}

	if cmd == "tailf" {
		if len(args) == 0 {
			return errors.New("no target provided, see --help")
		}
		cmdTailf := cmdTailf{Target: args[0]}
		return sm.DialRun(cmdTailf, network, address,
			sm.RawMode(),
			sm.ErrorAsEOF(),
			sm.WithLogger(gcLogger))
	}

	return fmt.Errorf("unknown command %s, see --help", cmd)
}
