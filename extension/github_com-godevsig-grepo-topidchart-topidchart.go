// Code generated by 'yaegi extract github.com/godevsig/grepo/topidchart/topidchart'. DO NOT EDIT.

//go:build topidchartmsg
// +build topidchartmsg

package extension

import (
	"github.com/godevsig/grepo/topidchart/topidchart"
	"reflect"
)

func init() {
	Symbols["github.com/godevsig/grepo/topidchart/topidchart/topidchart"] = map[string]reflect.Value{
		// type definitions
		"ProcessInfo":     reflect.ValueOf((*topidchart.ProcessInfo)(nil)),
		"Record":          reflect.ValueOf((*topidchart.Record)(nil)),
		"SessionRequest":  reflect.ValueOf((*topidchart.SessionRequest)(nil)),
		"SessionResponse": reflect.ValueOf((*topidchart.SessionResponse)(nil)),
		"SysInfo":         reflect.ValueOf((*topidchart.SysInfo)(nil)),
	}
}
