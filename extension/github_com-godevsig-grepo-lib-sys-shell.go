// Code generated by 'yaegi extract github.com/godevsig/grepo/lib/sys/shell'. DO NOT EDIT.

// +build shell

package extension

import (
	"github.com/godevsig/grepo/lib/sys/shell"
	"reflect"
)

func init() {
	Symbols["github.com/godevsig/grepo/lib/sys/shell/shell"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"New":     reflect.ValueOf(shell.New),
		"Run":     reflect.ValueOf(shell.Run),
		"RunWith": reflect.ValueOf(shell.RunWith),

		// type definitions
		"Shell": reflect.ValueOf((*shell.Shell)(nil)),
	}
}
