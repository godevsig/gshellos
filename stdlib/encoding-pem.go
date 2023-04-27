// Code generated by 'yaegi extract encoding/pem'. DO NOT EDIT.

//go:build go1.18 && !go1.19 && stdencoding
// +build go1.18,!go1.19,stdencoding

package stdlib

import (
	"encoding/pem"
	"reflect"
)

func init() {
	Symbols["encoding/pem/pem"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Decode":         reflect.ValueOf(pem.Decode),
		"Encode":         reflect.ValueOf(pem.Encode),
		"EncodeToMemory": reflect.ValueOf(pem.EncodeToMemory),

		// type definitions
		"Block": reflect.ValueOf((*pem.Block)(nil)),
	}
}
