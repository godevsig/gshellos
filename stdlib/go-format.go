// Code generated by 'yaegi extract go/format'. DO NOT EDIT.

//go:build go1.18 && !go1.19 && stdgo
// +build go1.18,!go1.19,stdgo

package stdlib

import (
	"go/format"
	"reflect"
)

func init() {
	Symbols["go/format/format"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Node":   reflect.ValueOf(format.Node),
		"Source": reflect.ValueOf(format.Source),
	}
}
