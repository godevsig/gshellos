// Code generated by 'yaegi extract embed'. DO NOT EDIT.

// +build go1.16,!go1.17,stdbase

package stdlib

import (
	"embed"
	"reflect"
)

func init() {
	Symbols["embed/embed"] = map[string]reflect.Value{
		// type definitions
		"FS": reflect.ValueOf((*embed.FS)(nil)),
	}
}