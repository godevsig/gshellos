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

const (
	usage = `gshell is an interpreter in Golang style syntax and a supervision
tool to manage .gsh apps.

SYNOPSIS
    gshell [OPTIONS] [COMMANDS]

Options:
    -c, --connect <hostname:port>
            Connect to the remote gRE server.

    -e, --gre <name>
            Specify the name of gRE(gshell Runtime Environment)
            in which the commands will be running.

            Gshell connects to the remote gRE server if -c provided,
            otherwise the local gRE server is used.

            The named gRE instance will be created before the
            execution of the following command if it has not been
            created by the remote/local gRE server.

    -d, --debug     Enable debug loglevel, -d -d makes more verbose log.
    -h, --help      Show this message.
    -v, --version   Show version information.

Commands:
    server [port]
            Start gRE(gshell Runtime Environment) server for local
            connection, also accept remote connection if [port]
            is provided.

    compile <file.gsh>
            Compile <file.gsh> into byte code <file>.

    exec <file[.gsh]> [args...]
            Run <file[.gsh]> in a new VM(virtual machine) in standalone
            mode.

    run [-i --rm] <file[.gsh]> [args...]
            Run <file[.gsh]> in a new VM(virtual machine) with its name
            set to base name of <file> in the designated gRE and return
            a 12 hex digits VM ID.

            -i    Enters interactive mode, keep STDIN and STDOUT
            open until <file[.gsh]> finishes execution.
            --rm  Automatically remove the VM when it exits.

            If no -c presents, the local gRE server is used.
            If no -e presents, the default "master" gRE is used.

Management Commands of gRE:
    ps
            List all VM instances in all gREs in the local/remote gRE server.
            If -e presents, only list the VMs in the designated gRE.

    kill <ID1 ID2 ...|name1 name2 ...>
            Abort the execution of one or more VMs in the designated gRE.

    rm <ID1 ID2 ...|name1 name2 ...>
            Remove one or more stopped VMs and associated files, running
            VM can not be removed.

    restart <ID1 ID2 ...|name1 name2 ...>
            Restart one or more stopped VMs in the designated gRE,
            no effect on a running VM.

    logs <server|gre|ID>
            Print the logs of the server or the designated gRE or the VM
            by ID so far.

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

	gsStream.SetLoglevel("*", loglevel)
	gsStream.SetFlag(logFlag)
	gcStream.SetLoglevel("*", loglevel)
	gcStream.SetFlag(logFlag)
	greStream.SetLoglevel("*", loglevel)
	greStream.SetFlag(logFlag)

	cmd := args[0]
	args = args[1:] // shift

	if cmd == "server" { // server [port]
		port := ""
		if len(args) > 0 {
			port = args[0]
		}
		return runServer(version, port)
	}

	if cmd == "rungre" { // rungre name
		var err error
		if len(args) > 0 {
			name := args[0]
			if name != "master" {
				err = rungre(name)
			}
		}
		return err
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
		return clientRun(cmdRun)
	}

	if cmd == "ps" {
		cmdPs := cmdPs{greName}
		return clientRun(cmdPs)
	}
	/*
		if cmd == "kill" {
			return greClient.kill()
		}
		if cmd == "outputs" {
			return greClient.outputs()
		}
		if cmd == "logs" {
			return greClient.logs()
		}
	*/
	return fmt.Errorf("unknown command %s, see --help", cmd)
}
