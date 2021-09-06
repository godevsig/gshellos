// Code generated by 'yaegi extract encoding/base32'. DO NOT EDIT.

// +build go1.16,!go1.17,stdencoding

package stdlib

import (
	"encoding/base32"
	"reflect"
)

func init() {
	Symbols["encoding/base32/base32"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"HexEncoding": reflect.ValueOf(&base32.HexEncoding).Elem(),
		"NewDecoder":  reflect.ValueOf(base32.NewDecoder),
		"NewEncoder":  reflect.ValueOf(base32.NewEncoder),
		"NewEncoding": reflect.ValueOf(base32.NewEncoding),
		"NoPadding":   reflect.ValueOf(base32.NoPadding),
		"StdEncoding": reflect.ValueOf(&base32.StdEncoding).Elem(),
		"StdPadding":  reflect.ValueOf(base32.StdPadding),

		// type definitions
		"CorruptInputError": reflect.ValueOf((*base32.CorruptInputError)(nil)),
		"Encoding":          reflect.ValueOf((*base32.Encoding)(nil)),
	}
}
