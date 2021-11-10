// Code generated by 'yaegi extract testing'. DO NOT EDIT.

// +build go1.16,!go1.17,stdtesting

package stdlib

import (
	"reflect"
	"testing"
)

func init() {
	Symbols["testing/testing"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AllocsPerRun":  reflect.ValueOf(testing.AllocsPerRun),
		"Benchmark":     reflect.ValueOf(testing.Benchmark),
		"CoverMode":     reflect.ValueOf(testing.CoverMode),
		"Coverage":      reflect.ValueOf(testing.Coverage),
		"Init":          reflect.ValueOf(testing.Init),
		"Main":          reflect.ValueOf(testing.Main),
		"MainStart":     reflect.ValueOf(testing.MainStart),
		"RegisterCover": reflect.ValueOf(testing.RegisterCover),
		"RunBenchmarks": reflect.ValueOf(testing.RunBenchmarks),
		"RunExamples":   reflect.ValueOf(testing.RunExamples),
		"RunTests":      reflect.ValueOf(testing.RunTests),
		"Short":         reflect.ValueOf(testing.Short),
		"Verbose":       reflect.ValueOf(testing.Verbose),

		// type definitions
		"B":                 reflect.ValueOf((*testing.B)(nil)),
		"BenchmarkResult":   reflect.ValueOf((*testing.BenchmarkResult)(nil)),
		"Cover":             reflect.ValueOf((*testing.Cover)(nil)),
		"CoverBlock":        reflect.ValueOf((*testing.CoverBlock)(nil)),
		"InternalBenchmark": reflect.ValueOf((*testing.InternalBenchmark)(nil)),
		"InternalExample":   reflect.ValueOf((*testing.InternalExample)(nil)),
		"InternalTest":      reflect.ValueOf((*testing.InternalTest)(nil)),
		"M":                 reflect.ValueOf((*testing.M)(nil)),
		"PB":                reflect.ValueOf((*testing.PB)(nil)),
		"T":                 reflect.ValueOf((*testing.T)(nil)),
		"TB":                reflect.ValueOf((*testing.TB)(nil)),

		// interface wrapper definitions
		"_TB": reflect.ValueOf((*_testing_TB)(nil)),
	}
}

// _testing_TB is an interface wrapper for TB type
type _testing_TB struct {
	IValue   interface{}
	WCleanup func(a0 func())
	WError   func(args ...interface{})
	WErrorf  func(format string, args ...interface{})
	WFail    func()
	WFailNow func()
	WFailed  func() bool
	WFatal   func(args ...interface{})
	WFatalf  func(format string, args ...interface{})
	WHelper  func()
	WLog     func(args ...interface{})
	WLogf    func(format string, args ...interface{})
	WName    func() string
	WSkip    func(args ...interface{})
	WSkipNow func()
	WSkipf   func(format string, args ...interface{})
	WSkipped func() bool
	WTempDir func() string
}

func (W _testing_TB) Cleanup(a0 func()) {
	W.WCleanup(a0)
}
func (W _testing_TB) Error(args ...interface{}) {
	W.WError(args...)
}
func (W _testing_TB) Errorf(format string, args ...interface{}) {
	W.WErrorf(format, args...)
}
func (W _testing_TB) Fail() {
	W.WFail()
}
func (W _testing_TB) FailNow() {
	W.WFailNow()
}
func (W _testing_TB) Failed() bool {
	return W.WFailed()
}
func (W _testing_TB) Fatal(args ...interface{}) {
	W.WFatal(args...)
}
func (W _testing_TB) Fatalf(format string, args ...interface{}) {
	W.WFatalf(format, args...)
}
func (W _testing_TB) Helper() {
	W.WHelper()
}
func (W _testing_TB) Log(args ...interface{}) {
	W.WLog(args...)
}
func (W _testing_TB) Logf(format string, args ...interface{}) {
	W.WLogf(format, args...)
}
func (W _testing_TB) Name() string {
	return W.WName()
}
func (W _testing_TB) Skip(args ...interface{}) {
	W.WSkip(args...)
}
func (W _testing_TB) SkipNow() {
	W.WSkipNow()
}
func (W _testing_TB) Skipf(format string, args ...interface{}) {
	W.WSkipf(format, args...)
}
func (W _testing_TB) Skipped() bool {
	return W.WSkipped()
}
func (W _testing_TB) TempDir() string {
	return W.WTempDir()
}
