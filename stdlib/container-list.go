// Code generated by 'yaegi extract container/list'. DO NOT EDIT.

//go:build go1.18 && !go1.19 && stdcontainer
// +build go1.18,!go1.19,stdcontainer

package stdlib

import (
	"container/list"
	"reflect"
)

func init() {
	Symbols["container/list/list"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"New": reflect.ValueOf(list.New),

		// type definitions
		"Element": reflect.ValueOf((*list.Element)(nil)),
		"List":    reflect.ValueOf((*list.List)(nil)),
	}
}
