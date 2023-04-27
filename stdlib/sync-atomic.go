// Code generated by 'yaegi extract sync/atomic'. DO NOT EDIT.

//go:build go1.18 && !go1.19 && stdbase
// +build go1.18,!go1.19,stdbase

package stdlib

import (
	"reflect"
	"sync/atomic"
)

func init() {
	Symbols["sync/atomic/atomic"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AddInt32":              reflect.ValueOf(atomic.AddInt32),
		"AddInt64":              reflect.ValueOf(atomic.AddInt64),
		"AddUint32":             reflect.ValueOf(atomic.AddUint32),
		"AddUint64":             reflect.ValueOf(atomic.AddUint64),
		"AddUintptr":            reflect.ValueOf(atomic.AddUintptr),
		"CompareAndSwapInt32":   reflect.ValueOf(atomic.CompareAndSwapInt32),
		"CompareAndSwapInt64":   reflect.ValueOf(atomic.CompareAndSwapInt64),
		"CompareAndSwapPointer": reflect.ValueOf(atomic.CompareAndSwapPointer),
		"CompareAndSwapUint32":  reflect.ValueOf(atomic.CompareAndSwapUint32),
		"CompareAndSwapUint64":  reflect.ValueOf(atomic.CompareAndSwapUint64),
		"CompareAndSwapUintptr": reflect.ValueOf(atomic.CompareAndSwapUintptr),
		"LoadInt32":             reflect.ValueOf(atomic.LoadInt32),
		"LoadInt64":             reflect.ValueOf(atomic.LoadInt64),
		"LoadPointer":           reflect.ValueOf(atomic.LoadPointer),
		"LoadUint32":            reflect.ValueOf(atomic.LoadUint32),
		"LoadUint64":            reflect.ValueOf(atomic.LoadUint64),
		"LoadUintptr":           reflect.ValueOf(atomic.LoadUintptr),
		"StoreInt32":            reflect.ValueOf(atomic.StoreInt32),
		"StoreInt64":            reflect.ValueOf(atomic.StoreInt64),
		"StorePointer":          reflect.ValueOf(atomic.StorePointer),
		"StoreUint32":           reflect.ValueOf(atomic.StoreUint32),
		"StoreUint64":           reflect.ValueOf(atomic.StoreUint64),
		"StoreUintptr":          reflect.ValueOf(atomic.StoreUintptr),
		"SwapInt32":             reflect.ValueOf(atomic.SwapInt32),
		"SwapInt64":             reflect.ValueOf(atomic.SwapInt64),
		"SwapPointer":           reflect.ValueOf(atomic.SwapPointer),
		"SwapUint32":            reflect.ValueOf(atomic.SwapUint32),
		"SwapUint64":            reflect.ValueOf(atomic.SwapUint64),
		"SwapUintptr":           reflect.ValueOf(atomic.SwapUintptr),

		// type definitions
		"Value": reflect.ValueOf((*atomic.Value)(nil)),
	}
}
