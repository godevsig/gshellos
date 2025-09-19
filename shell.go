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
	"regexp"
	"strings"
	"syscall"

	"github.com/godevsig/glib/sys/lined"
	"github.com/godevsig/gshellos/extension"
	"github.com/godevsig/gshellos/stdlib"
	"github.com/godevsig/gshellos/stdlib/unsafe"
	"github.com/traefik/yaegi/interp"
)

type gshell struct {
	pluginPath  string
	src         *sourceCode // nil for REPL
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

type sourceCode struct {
	codeDir string
	idFile  string
}

// unzip source code in required source code layout
func newSharedSourceCode(codeZip []byte) (*sourceCode, error) {
	hash := md5.Sum(codeZip)
	hashstr := hex.EncodeToString(hash[:])
	codeDir := filepath.Join(gshellTempDir, hashstr)
	idDir := filepath.Join(codeDir, ".IDHOLDER")
	idFile := filepath.Join(idDir, genID(4))
	src := &sourceCode{codeDir, idFile}

	if err := os.MkdirAll(idDir, 0755); err != nil {
		return nil, err
	}
	lockFile := filepath.Join(codeDir, ".lock")
	flock, err := os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer flock.Close()

	if err := syscall.Flock(int(flock.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	defer syscall.Flock(int(flock.Fd()), syscall.LOCK_UN)

	pkgDir := filepath.Join(codeDir, "src", "gshellmain")
	if _, err := os.Stat(pkgDir); err != nil {
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

	// place an empty file inside to prevent early removal of codeDir
	if file, err := os.Create(idFile); err == nil {
		file.Close()
	}

	return src, nil
}

func (src *sourceCode) close() {
	os.Remove(src.idFile)
	entries, _ := os.ReadDir(filepath.Dir(src.idFile))
	if len(entries) == 0 {
		os.RemoveAll(src.codeDir)
	}
}

func newShellWithCodeZip(codeZip []byte, pluginPath string) (*gshell, error) {
	gsh := &gshell{}
	if codeZip != nil {
		src, err := newSharedSourceCode(codeZip)
		if err != nil {
			return nil, err
		}
		gsh.src = src
	}
	gsh.pluginPath = pluginPath

	return gsh, nil
}

func newShell(pluginPath string) (*gshell, error) {
	return newShellWithCodeZip(nil, pluginPath)
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
	if gsh.src == nil {
		i.ImportUsed()
	}
	if err := i.Use(extension.BuiltinSymbols); err != nil {
		return err
	}
	if err := i.Use(extension.PluginSymbols); err != nil {
		return err
	}
	os.Args = opt.Args //reset os.Args for interpreter
	gsh.interpreter = i
	if gsh.src != nil {
		if _, err := i.Eval(`import "gshellmain"`); err != nil {
			return err
		}
	}
	return nil
}

func moduleNameToFileName(path string) string {
	path = strings.ReplaceAll(path, ".", "_")
	path = strings.ReplaceAll(path, "/", "-")
	return path
}

func (gsh *gshell) tryLoadMissingPlugin(origErr error) error {
	re := regexp.MustCompile(`import "([^"]+)" error:`)
	matches := re.FindAllStringSubmatch(origErr.Error(), -1)
	if len(matches) == 0 {
		return origErr
	}
	// last import path
	module := matches[len(matches)-1][1]

	pluginFile := filepath.Join(gsh.pluginPath, moduleNameToFileName(module)+".gp")
	exports, loadErr := loadPluginFile(pluginFile)
	if loadErr != nil {
		return fmt.Errorf("%v: %v", loadErr, origErr)
	}

	if gsh.src == nil {
		if useErr := gsh.interpreter.Use(exports); useErr != nil {
			return fmt.Errorf("%v: %v", useErr, origErr)
		}
	}

	return nil
}

func (gsh *gshell) initWithPlugin(opt interp.Options) error {
	for {
		if err := gsh.init(opt); err != nil {
			if err := gsh.tryLoadMissingPlugin(err); err != nil {
				return err
			}
			continue // retry after successful plugin load
		}
		return nil
	}
}

func (gsh *gshell) start(ctx context.Context) error {
	_, err := gsh.interpreter.EvalWithContext(ctx, "gshellmain.main()")
	return err
}

func (gsh *gshell) stop() error {
	_, err := gsh.interpreter.Eval("gshellmain.Stop()")
	return err
}

func (gsh *gshell) abort(cancel context.CancelFunc) {
	cancel()
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
		if len(line) == 0 {
			continue
		}
		if _, err := gsh.interpreter.EvalWithContext(ctx, line); err != nil {
			if err := gsh.tryLoadMissingPlugin(err); err != nil {
				fmt.Println(err)
				continue
			}
			if _, err := gsh.interpreter.EvalWithContext(ctx, line); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func init() {
	os.Setenv("YAEGI_SPECIAL_STDIO", "1")
}
