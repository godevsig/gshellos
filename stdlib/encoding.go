// Code generated by 'yaegi extract encoding'. DO NOT EDIT.

// +build go1.16,!go1.17,stdencoding

package stdlib

import (
	"encoding"
	"reflect"
)

func init() {
	Symbols["encoding/encoding"] = map[string]reflect.Value{
		// type definitions
		"BinaryMarshaler":   reflect.ValueOf((*encoding.BinaryMarshaler)(nil)),
		"BinaryUnmarshaler": reflect.ValueOf((*encoding.BinaryUnmarshaler)(nil)),
		"TextMarshaler":     reflect.ValueOf((*encoding.TextMarshaler)(nil)),
		"TextUnmarshaler":   reflect.ValueOf((*encoding.TextUnmarshaler)(nil)),

		// interface wrapper definitions
		"_BinaryMarshaler":   reflect.ValueOf((*_encoding_BinaryMarshaler)(nil)),
		"_BinaryUnmarshaler": reflect.ValueOf((*_encoding_BinaryUnmarshaler)(nil)),
		"_TextMarshaler":     reflect.ValueOf((*_encoding_TextMarshaler)(nil)),
		"_TextUnmarshaler":   reflect.ValueOf((*_encoding_TextUnmarshaler)(nil)),
	}
}

// _encoding_BinaryMarshaler is an interface wrapper for BinaryMarshaler type
type _encoding_BinaryMarshaler struct {
	IValue         interface{}
	WMarshalBinary func() (data []byte, err error)
}

func (W _encoding_BinaryMarshaler) MarshalBinary() (data []byte, err error) {
	if W.WMarshalBinary == nil {
		return
	}
	return W.WMarshalBinary()
}

// _encoding_BinaryUnmarshaler is an interface wrapper for BinaryUnmarshaler type
type _encoding_BinaryUnmarshaler struct {
	IValue           interface{}
	WUnmarshalBinary func(data []byte) (r0 error)
}

func (W _encoding_BinaryUnmarshaler) UnmarshalBinary(data []byte) (r0 error) {
	if W.WUnmarshalBinary == nil {
		return
	}
	return W.WUnmarshalBinary(data)
}

// _encoding_TextMarshaler is an interface wrapper for TextMarshaler type
type _encoding_TextMarshaler struct {
	IValue       interface{}
	WMarshalText func() (text []byte, err error)
}

func (W _encoding_TextMarshaler) MarshalText() (text []byte, err error) {
	if W.WMarshalText == nil {
		return
	}
	return W.WMarshalText()
}

// _encoding_TextUnmarshaler is an interface wrapper for TextUnmarshaler type
type _encoding_TextUnmarshaler struct {
	IValue         interface{}
	WUnmarshalText func(text []byte) (r0 error)
}

func (W _encoding_TextUnmarshaler) UnmarshalText(text []byte) (r0 error) {
	if W.WUnmarshalText == nil {
		return
	}
	return W.WUnmarshalText(text)
}
