package gshellos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/godevsig/glib/sys/lined"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/stdlib"
	"github.com/traefik/yaegi/interp"
)

type gshell struct {
	codeDir     string
	interpreter *interp.Interpreter
}

func newShell(opt interp.Options) (*gshell, error) {
	gsh := &gshell{}
	tmpDir, err := os.MkdirTemp(gshellTempDir, "code-")
	if err != nil {
		return nil, err
	}
	gsh.codeDir = tmpDir
	opt.GoPath = tmpDir
	i := interp.New(opt)
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}
	if err := i.Use(extension.Symbols); err != nil {
		return nil, err
	}
	i.ImportUsed()
	gsh.interpreter = i
	os.Args = opt.Args //reset os.Args for interpreter

	return gsh, nil
}

func (gsh *gshell) close() {
	os.RemoveAll(gsh.codeDir)
	gsh.interpreter = nil
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
