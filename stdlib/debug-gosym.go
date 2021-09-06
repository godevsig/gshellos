// Code generated by 'yaegi extract debug/gosym'. DO NOT EDIT.

// +build go1.16,!go1.17,stddebug

package stdlib

import (
	"debug/gosym"
	"reflect"
)

func init() {
	Symbols["debug/gosym/gosym"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"NewLineTable": reflect.ValueOf(gosym.NewLineTable),
		"NewTable":     reflect.ValueOf(gosym.NewTable),

		// type definitions
		"DecodingError":    reflect.ValueOf((*gosym.DecodingError)(nil)),
		"Func":             reflect.ValueOf((*gosym.Func)(nil)),
		"LineTable":        reflect.ValueOf((*gosym.LineTable)(nil)),
		"Obj":              reflect.ValueOf((*gosym.Obj)(nil)),
		"Sym":              reflect.ValueOf((*gosym.Sym)(nil)),
		"Table":            reflect.ValueOf((*gosym.Table)(nil)),
		"UnknownFileError": reflect.ValueOf((*gosym.UnknownFileError)(nil)),
		"UnknownLineError": reflect.ValueOf((*gosym.UnknownLineError)(nil)),
	}
}
