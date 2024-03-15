package gshellos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/godevsig/glib/sys/lined"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/stdlib"
	"github.com/godevsig/gshellos/stdlib/unsafe"
	"github.com/traefik/yaegi/interp"
)

type gshell struct {
	codeDir     string
	interpreter *interp.Interpreter
}

func mainPkgToGshellPkg(path string) error {
	modifyGoFile := func(file string, perm fs.FileMode) error {
		buf, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		before, after, found := bytes.Cut(buf, []byte("package main"))
		if found && len(before) == 0 {
			return os.WriteFile(file, append([]byte("package gshellmain"), after...), perm)
		}
		return nil
	}

	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode().IsRegular() && strings.HasSuffix(fi.Name(), ".go") {
		return modifyGoFile(path, fi.Mode().Perm())
	}

	entires, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entires {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".go") {
			if err := modifyGoFile(filepath.Join(path, entry.Name()), entry.Type().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

// unzip source code into codeDir in required source code layout
func prepareSourceCode(codeDir string, codeZip []byte) error {
	pkgDir := filepath.Join(codeDir, "src", "gshellmain")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	if err := unzipBufferToPath(codeZip, pkgDir); err != nil {
		//os.RemoveAll(codeDir)
		return err
	}
	if err := mainPkgToGshellPkg(pkgDir); err != nil {
		//os.RemoveAll(codeDir)
		return err
	}
	return nil
}

// new an interpreter, with GOPATH set to codeDir, which might have been
// prepared by prepareSourceCode()
func newShell(codeDir string, opt interp.Options) (*gshell, error) {
	gsh := &gshell{}
	gsh.codeDir = codeDir
	opt.GoPath = codeDir
	i := interp.New(opt)
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}
	if err := i.Use(unsafe.Symbols); err != nil {
		return nil, err
	}
	if err := i.Use(extension.Symbols); err != nil {
		return nil, err
	}
	i.ImportUsed()
	os.Args = opt.Args //reset os.Args for interpreter
	gsh.interpreter = i
	var err error
	if codeDir != "" {
		_, err = i.Eval(`import "gshellmain"`)
	}
	return gsh, err
}

func (gsh *gshell) close() {
	//os.RemoveAll(gsh.codeDir)
	gsh.interpreter = nil
}

func (gsh *gshell) start(ctx context.Context) (err error) {
	_, err = gsh.interpreter.EvalWithContext(ctx, "gshellmain.main()")
	return
}

func (gsh *gshell) stop(cancel context.CancelFunc) {
	if _, err := gsh.interpreter.Eval("gshellmain.Stop()"); err != nil {
		cancel()
	}
}

func (gsh *gshell) evalPath(path string) error {
	return gsh.evalPathWithContext(nil, path)
}

func (gsh *gshell) evalPathWithContext(ctx context.Context, path string) error {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	dir := path
	file := ""
	if fi.Mode().IsRegular() {
		dir = filepath.Dir(path)
		file = filepath.Base(path)
		if !strings.HasSuffix(file, ".go") {
			return errors.New("wrong file suffix, .go expected")
		}
	}

	srcPath := filepath.Join(gsh.codeDir, "src")
	if file != "" {
		if err := os.Symlink(dir, srcPath); err != nil {
			return err
		}
		srcPath = filepath.Join(srcPath, file)
	} else {
		if err := os.MkdirAll(srcPath, 0755); err != nil {
			return err
		}
		if err := os.Symlink(dir, filepath.Join(srcPath, "vendor")); err != nil {
			return err
		}
		srcPath = "."
	}

	if ctx == nil {
		_, err = gsh.interpreter.EvalPath(srcPath)
	} else {
		_, err = gsh.interpreter.EvalPathWithContext(ctx, srcPath)
	}

	return err
}

func (gsh *gshell) runREPL() {
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
			_, err := gsh.interpreter.EvalWithContext(ctx, line)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func init() {
	os.Setenv("YAEGI_SPECIAL_STDIO", "1")
}
