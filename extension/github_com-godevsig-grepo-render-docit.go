// Code generated by 'yaegi extract github.com/godevsig/grepo/render/docit'. DO NOT EDIT.

// +build docit

package extension

import (
	"github.com/godevsig/grepo/render/docit"
	"reflect"
)

func init() {
	Symbols["github.com/godevsig/grepo/render/docit/docit"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"NewServer": reflect.ValueOf(docit.NewServer),

		// type definitions
		"HTMLResponse":    reflect.ValueOf((*docit.HTMLResponse)(nil)),
		"MarkdownRequest": reflect.ValueOf((*docit.MarkdownRequest)(nil)),
		"Server":          reflect.ValueOf((*docit.Server)(nil)),
	}
}
