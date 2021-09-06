// Code generated by 'yaegi extract go/parser'. DO NOT EDIT.

// +build go1.16,!go1.17,stdgo

package stdlib

import (
	"go/parser"
	"reflect"
)

func init() {
	Symbols["go/parser/parser"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AllErrors":         reflect.ValueOf(parser.AllErrors),
		"DeclarationErrors": reflect.ValueOf(parser.DeclarationErrors),
		"ImportsOnly":       reflect.ValueOf(parser.ImportsOnly),
		"PackageClauseOnly": reflect.ValueOf(parser.PackageClauseOnly),
		"ParseComments":     reflect.ValueOf(parser.ParseComments),
		"ParseDir":          reflect.ValueOf(parser.ParseDir),
		"ParseExpr":         reflect.ValueOf(parser.ParseExpr),
		"ParseExprFrom":     reflect.ValueOf(parser.ParseExprFrom),
		"ParseFile":         reflect.ValueOf(parser.ParseFile),
		"SpuriousErrors":    reflect.ValueOf(parser.SpuriousErrors),
		"Trace":             reflect.ValueOf(parser.Trace),

		// type definitions
		"Mode": reflect.ValueOf((*parser.Mode)(nil)),
	}
}
