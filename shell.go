package gshellos

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/godevsig/glib/sys/lined"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/stdlib"
	"github.com/godevsig/gshellos/stdlib/unsafe"
	"github.com/traefik/yaegi/interp"
)

type gshell struct {
	src         *sourceCode
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
	if fi.Mode().IsRegular() {
		if !strings.HasSuffix(fi.Name(), ".go") {
			return errors.New("wrong file suffix, .go expected")
		}
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

var (
	sharedCodeDir     = make(map[string]int) // [codeDir]refCnt
	sharedCodeDirLock sync.Mutex
)

type sourceCode struct {
	codeDir string
}

// unzip source code in required source code layout
func newSharedSourceCode(codeZip []byte) (src *sourceCode, err error) {
	hash := md5.Sum(codeZip)
	hashstr := hex.EncodeToString(hash[:])
	codeDir := filepath.Join(gshellTempDir, hashstr)
	src = &sourceCode{codeDir}

	sharedCodeDirLock.Lock()
	defer sharedCodeDirLock.Unlock()
	refCnt := sharedCodeDir[codeDir]
	if refCnt == 0 {
		defer func() {
			if err != nil {
				os.RemoveAll(codeDir)
			}
		}()
		pkgDir := filepath.Join(codeDir, "src", "gshellmain")
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return src, err
		}
		if err := unzipBufferToPath(codeZip, pkgDir); err != nil {
			return src, err
		}
		if err := mainPkgToGshellPkg(pkgDir); err != nil {
			return src, err
		}
	}
	sharedCodeDir[codeDir] = refCnt + 1
	return src, nil
}

func (src *sourceCode) close() {
	sharedCodeDirLock.Lock()
	defer sharedCodeDirLock.Unlock()
	refCnt := sharedCodeDir[src.codeDir] - 1
	if refCnt == 0 {
		os.RemoveAll(src.codeDir)
	}
}

func newShellWithCodeZip(codeZip []byte) (*gshell, error) {
	gsh := &gshell{}
	if codeZip != nil {
		src, err := newSharedSourceCode(codeZip)
		if err != nil {
			return nil, err
		}
		gsh.src = src
	}

	return gsh, nil
}

func newShell() (*gshell, error) {
	return newShellWithCodeZip(nil)
}

func (gsh *gshell) close() {
	if gsh.src != nil {
		gsh.src.close()
	}
	gsh.interpreter = nil
}

func (gsh *gshell) init(opt interp.Options) error {
	if gsh.src != nil {
		opt.GoPath = gsh.src.codeDir
	}

	i := interp.New(opt)
	if err := i.Use(stdlib.Symbols); err != nil {
		return err
	}
	if err := i.Use(unsafe.Symbols); err != nil {
		return err
	}
	if err := i.Use(extension.Symbols); err != nil {
		return err
	}
	i.ImportUsed()
	os.Args = opt.Args //reset os.Args for interpreter
	gsh.interpreter = i
	if gsh.src != nil {
		if _, err := i.Eval(`import "gshellmain"`); err != nil {
			return err
		}
	}
	return nil
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
