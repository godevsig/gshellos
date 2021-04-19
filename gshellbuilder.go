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
)

var version string

const (
	usage = `gshell is shell alike interpreter based on Golang

SYNOPSIS
	gshell [options] [file[.gsh]]

OPTIONS
	-h, --help        show this message
	-v, --version     show version information
	-c, --compile     compile .gsh file

gshell runs the file with .gsh suffix, or compiles the file, or
directly executes the compiled file, or
enters interactive mode if no file supplied.

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

// ToDo: add new scope ScopeExtend in upstream tengo project
//symbol.Scope = ScopeExtend
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
	sh.preCompile()

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

func (sh *shell) compileAndRun(inputFile string) (err error) {
	bytecode, err := sh.compile(inputFile)
	if err != nil {
		return
	}

	machine := tengo.NewVM(bytecode, sh.globals, -1)
	err = machine.Run()
	return
}

func (sh *shell) runCompiled(inputFile string) (err error) {
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

// ShellMain is the main entry of gshell
func ShellMain() error {
	args := os.Args
	sh := &shell{}
	sh.initModules()

	// no arg, shell mode
	if len(args) == 1 {
		sh.runREPL()
		return nil
	}

	compileOnly := false
	// file or option
	switch args[1] {
	case "-h", "--help":
		fmt.Println(usage)
	case "-v", "--version":
		if len(version) == 0 {
			version = "development"
		}
		fmt.Println(version)
	case "-c", "--compile":
		compileOnly = true
		args = args[1:] // shift
		if len(args) == 1 {
			return errors.New("no file provided, see --help")
		}
		if filepath.Ext(args[1]) != ".gsh" {
			return errors.New("wrong file suffix, see --help")
		}
		fallthrough
	default:
		inputFile, _ := filepath.Abs(args[1])
		sh.preCompile()

		if compileOnly {
			outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
			err := sh.compileAndSave(inputFile, outputFile)
			return err
		}

		args = args[1:] // shift
		os.Args = args  // pass os.Args down
		if filepath.Ext(inputFile) == ".gsh" {
			err := sh.compileAndRun(inputFile)
			return err
		}

		err := sh.runCompiled(inputFile)
		return err
	}
	return nil
}
