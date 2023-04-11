package gshellos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/godevsig/grepo/lib/sys/lined"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/stdlib"
	"github.com/traefik/yaegi/interp"
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

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsDir()
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

	if isDir(path) {
		_, err := sh.EvalPath(path)
		return err
	}

	return fmt.Errorf("%s not found", path)
}
