// Package shell lets user call shell commands from tengo scripts.
// Try below:
//  sh := import("shell")
//  fmt := import("fmt")
//
//  fmt.print(sh.run(`ls -l`).output())
//  cdr := sh.run(`ping localhost`)
//  fmt.println(cdr.wait(5))
//  cdr.kill()
//  cdr.wait() // or cdr.wait(0)
//  fmt.println(cdr.errcode())
//  fmt.println(cdr.output())
//  fmt.println(sh.run(`touch /etc/fstab`).output())
package shell

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/d5/tengo/v2"
	gs "github.com/godevsig/gshellos"
)

func init() {
	gs.RegModule("shell", module)
}

var module = map[string]tengo.Object{
	"run": &gs.UserFunction{
		Value:     run,
		Signature: `run(cmd string) -> commander`,
		Usage:     `Run shell command cmd and return a commander obj`,
		Example:   `cdr := sh.run("ls -lh | grep go")`,
	},
}

type commander struct {
	*exec.Cmd
	waitChan chan error
	err      error
	done     bool
}

func newCommander(input string) *commander {
	if len(input) == 0 {
		return nil
	}
	cmd := exec.Command("bash", "-c", input)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return &commander{Cmd: cmd, waitChan: make(chan error)}
}

// Return true if cdr is done
func (cdr *commander) wait(seconds int64) bool {
	if cdr.done {
		return true
	}

	if seconds <= 0 {
		seconds = 3153600000 // 100 years
	}

	select {
	case cdr.err = <-cdr.waitChan:
		if cdr.err != nil {
			//cdr.Stdout.Write([]byte("\n"))
			cdr.Stdout.Write(cdr.Stderr.(*bytes.Buffer).Bytes())
			//cdr.Stdout.Write([]byte("\n"))
			//cdr.Stdout.Write([]byte(cdr.err.Error()))
		}
		cdr.done = true
	case <-time.After(time.Duration(seconds) * time.Second):
		return false
	}

	return true
}

// Get the output of the command after its execution is done.
// The output will have content of stderr and return code if error happened.
func (cdr *commander) output(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 0 {
		return nil, tengo.ErrWrongNumArguments
	}
	cdr.wait(-1)

	return gs.ToObject(cdr.Stdout.(*bytes.Buffer).String())
}

// Errcode is the returned exit code of the command.
//	0: the command successfully did the job.
//	-1: the command was killed by kill().
//	>0: the command itself failed.
//	-99: undefined error.
func (cdr *commander) errcode(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 0 {
		return nil, tengo.ErrWrongNumArguments
	}
	cdr.wait(-1)

	var rc int64
	if cdr.err != nil {
		switch v := cdr.err.(type) {
		case *exec.ExitError:
			rc = int64(v.ProcessState.ExitCode())
		default:
			rc = -99
		}
	}
	return gs.ToObject(rc)
}

// Wait the command to complete.
// Wait can have optional timeout if the first arg is int.
// Wait forever if the optional timeout not specified, or timeout <= 0
// Return true if the commander exited(successfully or not) within the timeout peroid.
func (cdr *commander) waitTimeout(args ...tengo.Object) (tengo.Object, error) {
	if len(args) > 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	timeOut := -1
	if len(args) == 1 {
		if err := gs.FromObject(&timeOut, args[0]); err != nil {
			return nil, err
		}
	}

	if cdr.wait(int64(timeOut)) {
		return tengo.TrueValue, nil
	}
	return tengo.FalseValue, nil
}

// Run a sh command/script in background, e.g.
// cmd := sh.run(`ls -l`)
func run(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	var input string
	if err := gs.FromObject(&input, args[0]); err != nil {
		return nil, err
	}

	cmd := newCommander(input)
	if cmd == nil {
		return nil, errors.New("unexpected empty command in sh.run()")
	}

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start cmd %s failed: %w", cmd.String(), err)
	}

	go func() { cmd.waitChan <- cmd.Wait() }()

	return commanderObj(cmd), nil
}

// Kill the underlying process held by the commander.
//
// Kill causes the Process to exit immediately.
// Kill does not wait until the Process has actually exited.
// This only kills the Process itself, not any other processes it may have started.
func (cdr *commander) kill(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 0 {
		return nil, tengo.ErrWrongNumArguments
	}

	if cdr.done {
		return nil, nil
	}

	cdr.Process.Kill()
	return nil, nil
}

func commanderObj(cdr *commander) tengo.Object {
	obj := map[string]tengo.Object{
		"output": &gs.UserFunction{
			Value:     cdr.output,
			Signature: `output() -> string`,
			Usage:     `Get the output of the commander obj, will bock until commander exits.`,
			Example:   `fmt.println(cdr.output())`,
		},
		"wait": &gs.UserFunction{
			Value: cdr.waitTimeout,
			Signature: `wait() -> bool
					wait(timeout int) -> bool`,
			Usage: `Block timeout seconds to wait the commander to complete, wait forever if no timeout provided.
					Return true if the commander exits within the timeout peroid.`,
			Example: `if cdr.wait(3) { fmt.println("commander exits within 3 seconds.") }`,
		},
		"errcode": &gs.UserFunction{
			Value:     cdr.errcode,
			Signature: `errcode() -> int`,
			Usage:     `Return error code of the commander, will bock until commander exits.`,
			Example:   `fmt.println(cdr.errcode())`,
		},
		"kill": &gs.UserFunction{
			Value:     cdr.kill,
			Signature: `kill()`,
			Usage:     `Kill the commander.`,
			Example:   `cdr.kill()`,
		},
	}
	return gs.MustToObject(obj)
}
