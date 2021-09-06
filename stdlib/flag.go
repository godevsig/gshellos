// Code generated by 'yaegi extract flag'. DO NOT EDIT.

// +build go1.16,!go1.17,stdbase

package stdlib

import (
	"flag"
	"reflect"
)

func init() {
	Symbols["flag/flag"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Arg":             reflect.ValueOf(flag.Arg),
		"Args":            reflect.ValueOf(flag.Args),
		"Bool":            reflect.ValueOf(flag.Bool),
		"BoolVar":         reflect.ValueOf(flag.BoolVar),
		"CommandLine":     reflect.ValueOf(&flag.CommandLine).Elem(),
		"ContinueOnError": reflect.ValueOf(flag.ContinueOnError),
		"Duration":        reflect.ValueOf(flag.Duration),
		"DurationVar":     reflect.ValueOf(flag.DurationVar),
		"ErrHelp":         reflect.ValueOf(&flag.ErrHelp).Elem(),
		"ExitOnError":     reflect.ValueOf(flag.ExitOnError),
		"Float64":         reflect.ValueOf(flag.Float64),
		"Float64Var":      reflect.ValueOf(flag.Float64Var),
		"Func":            reflect.ValueOf(flag.Func),
		"Int":             reflect.ValueOf(flag.Int),
		"Int64":           reflect.ValueOf(flag.Int64),
		"Int64Var":        reflect.ValueOf(flag.Int64Var),
		"IntVar":          reflect.ValueOf(flag.IntVar),
		"Lookup":          reflect.ValueOf(flag.Lookup),
		"NArg":            reflect.ValueOf(flag.NArg),
		"NFlag":           reflect.ValueOf(flag.NFlag),
		"NewFlagSet":      reflect.ValueOf(flag.NewFlagSet),
		"PanicOnError":    reflect.ValueOf(flag.PanicOnError),
		"Parse":           reflect.ValueOf(flag.Parse),
		"Parsed":          reflect.ValueOf(flag.Parsed),
		"PrintDefaults":   reflect.ValueOf(flag.PrintDefaults),
		"Set":             reflect.ValueOf(flag.Set),
		"String":          reflect.ValueOf(flag.String),
		"StringVar":       reflect.ValueOf(flag.StringVar),
		"Uint":            reflect.ValueOf(flag.Uint),
		"Uint64":          reflect.ValueOf(flag.Uint64),
		"Uint64Var":       reflect.ValueOf(flag.Uint64Var),
		"UintVar":         reflect.ValueOf(flag.UintVar),
		"UnquoteUsage":    reflect.ValueOf(flag.UnquoteUsage),
		"Usage":           reflect.ValueOf(&flag.Usage).Elem(),
		"Var":             reflect.ValueOf(flag.Var),
		"Visit":           reflect.ValueOf(flag.Visit),
		"VisitAll":        reflect.ValueOf(flag.VisitAll),

		// type definitions
		"ErrorHandling": reflect.ValueOf((*flag.ErrorHandling)(nil)),
		"Flag":          reflect.ValueOf((*flag.Flag)(nil)),
		"FlagSet":       reflect.ValueOf((*flag.FlagSet)(nil)),
		"Getter":        reflect.ValueOf((*flag.Getter)(nil)),
		"Value":         reflect.ValueOf((*flag.Value)(nil)),

		// interface wrapper definitions
		"_Getter": reflect.ValueOf((*_flag_Getter)(nil)),
		"_Value":  reflect.ValueOf((*_flag_Value)(nil)),
	}
}

// _flag_Getter is an interface wrapper for Getter type
type _flag_Getter struct {
	IValue  interface{}
	WGet    func() interface{}
	WSet    func(a0 string) error
	WString func() string
}

func (W _flag_Getter) Get() interface{}    { return W.WGet() }
func (W _flag_Getter) Set(a0 string) error { return W.WSet(a0) }
func (W _flag_Getter) String() string      { return W.WString() }

// _flag_Value is an interface wrapper for Value type
type _flag_Value struct {
	IValue  interface{}
	WSet    func(a0 string) error
	WString func() string
}

func (W _flag_Value) Set(a0 string) error { return W.WSet(a0) }
func (W _flag_Value) String() string      { return W.WString() }
