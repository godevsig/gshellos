// Package lined is line editor, supports operations on terminal.
//
// Supported operations includes short-cut like Ctrl-A Ctrl-E Ctrl-C Ctrl-D...
// and UP/DOWN for history.
package lined

import (
	"strings"

	"github.com/peterh/liner"
)

// Cfg is the options that lined supports, used when New the Editor instance.
type Cfg struct {
	Prompt string
}

// Editor is the instance of a line editor
type Editor struct {
	lnr *liner.State
	cfg Cfg
}

// NewEditor returns an instance of Editor, with user defined Cfg.
func NewEditor(cfg Cfg) *Editor {
	lnr := liner.NewLiner()
	return &Editor{lnr, cfg}
}

// Readline read and return a line from input, with leading and trailing white space removed.
// An io.EOF error is returned if user entered Ctrl-D
func (ed *Editor) Readline() (line string, err error) {
	line, err = ed.lnr.Prompt(ed.cfg.Prompt)
	if len(line) != 0 {
		line = strings.TrimSpace(line)
		ed.lnr.AppendHistory(line)
	}
	return
}

// Close returns the terminal to its previous mode.
func (ed *Editor) Close() {
	ed.lnr.Close()
}
