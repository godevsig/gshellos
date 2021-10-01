// Code generated by 'yaegi extract net/http/cookiejar'. DO NOT EDIT.

// +build go1.16,!go1.17,stdhttp

package stdlib

import (
	"net/http/cookiejar"
	"reflect"
)

func init() {
	Symbols["net/http/cookiejar/cookiejar"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"New": reflect.ValueOf(cookiejar.New),

		// type definitions
		"Jar":              reflect.ValueOf((*cookiejar.Jar)(nil)),
		"Options":          reflect.ValueOf((*cookiejar.Options)(nil)),
		"PublicSuffixList": reflect.ValueOf((*cookiejar.PublicSuffixList)(nil)),

		// interface wrapper definitions
		"_PublicSuffixList": reflect.ValueOf((*_net_http_cookiejar_PublicSuffixList)(nil)),
	}
}

// _net_http_cookiejar_PublicSuffixList is an interface wrapper for PublicSuffixList type
type _net_http_cookiejar_PublicSuffixList struct {
	IValue        interface{}
	WPublicSuffix func(domain string) (r0 string)
	WString       func() (r0 string)
}

func (W _net_http_cookiejar_PublicSuffixList) PublicSuffix(domain string) (r0 string) {
	if W.WPublicSuffix == nil {
		return
	}
	return W.WPublicSuffix(domain)
}
func (W _net_http_cookiejar_PublicSuffixList) String() (r0 string) {
	if W.WString == nil {
		return
	}
	return W.WString()
}
