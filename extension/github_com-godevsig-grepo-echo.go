// Code generated by 'yaegi extract github.com/godevsig/grepo/echo'. DO NOT EDIT.

//go:build echo
// +build echo

package extension

import (
	"github.com/godevsig/grepo/echo"
	"go/constant"
	"go/token"
	"reflect"
)

func init() {
	Symbols["github.com/godevsig/grepo/echo/echo"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"NewServer":   reflect.ValueOf(echo.NewServer),
		"Publisher":   reflect.ValueOf(constant.MakeFromLiteral("\"example\"", token.STRING, 0)),
		"ServiceEcho": reflect.ValueOf(constant.MakeFromLiteral("\"echo.v1.0\"", token.STRING, 0)),

		// type definitions
		"Reply":           reflect.ValueOf((*echo.Reply)(nil)),
		"Request":         reflect.ValueOf((*echo.Request)(nil)),
		"Server":          reflect.ValueOf((*echo.Server)(nil)),
		"SubWhoElseEvent": reflect.ValueOf((*echo.SubWhoElseEvent)(nil)),
		"WhoElse":         reflect.ValueOf((*echo.WhoElse)(nil)),
	}
}
func init() {
	Symbols["github.com/godevsig/grepo/echo/echo/echo"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Publisher":   reflect.ValueOf(constant.MakeFromLiteral("\"example\"", token.STRING, 0)),
		"ServiceEcho": reflect.ValueOf(constant.MakeFromLiteral("\"echo.v1.0\"", token.STRING, 0)),

		// type definitions
		"Reply":           reflect.ValueOf((*echo.Reply)(nil)),
		"Request":         reflect.ValueOf((*echo.Request)(nil)),
		"SubWhoElseEvent": reflect.ValueOf((*echo.SubWhoElseEvent)(nil)),
		"WhoElse":         reflect.ValueOf((*echo.WhoElse)(nil)),
	}
}
