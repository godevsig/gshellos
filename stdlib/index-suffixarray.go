// Code generated by 'yaegi extract index/suffixarray'. DO NOT EDIT.

// +build go1.16,!go1.17,stdalgorithm

package stdlib

import (
	"index/suffixarray"
	"reflect"
)

func init() {
	Symbols["index/suffixarray/suffixarray"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"New": reflect.ValueOf(suffixarray.New),

		// type definitions
		"Index": reflect.ValueOf((*suffixarray.Index)(nil)),
	}
}