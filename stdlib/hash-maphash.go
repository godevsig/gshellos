// Code generated by 'yaegi extract hash/maphash'. DO NOT EDIT.

// +build go1.16,!go1.17,stdhash

package stdlib

import (
	"hash/maphash"
	"reflect"
)

func init() {
	Symbols["hash/maphash/maphash"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"MakeSeed": reflect.ValueOf(maphash.MakeSeed),

		// type definitions
		"Hash": reflect.ValueOf((*maphash.Hash)(nil)),
		"Seed": reflect.ValueOf((*maphash.Seed)(nil)),
	}
}
