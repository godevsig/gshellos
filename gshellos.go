package gshellos

import (
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/d5/tengo/v2"
)

// List of symbol scopes
const (
	ScopeExtend tengo.SymbolScope = "EXTEND"
)

var (
	// ErrNotExtendable is an error where an Object is not extendable.
	ErrNotExtendable = errors.New("not extendable")
	// ErrUndefinedMethod is an error when calling an unknown method.
	ErrUndefinedMethod = errors.New("method not defined")
	// ErrInvalidPtr is an error where the input is not a valid pointer.
	ErrInvalidPtr = errors.New("invalid or nil pointer")
	// ErrInvalidType is an error where the input type is not expected.
	ErrInvalidType = errors.New("invalid type")
	// ErrNotConvertible is an error when failed to convert the input to/from an object.
	ErrNotConvertible = errors.New("not convertible")
)

// modules are extension modules managed by this package.
var modules = map[string]map[string]tengo.Object{}

func init() {
	gob.Register(&UserFunction{})
}

// RegModule should be called in init() func in the to be registered extension module.
func RegModule(name string, mod map[string]tengo.Object) {
	if _, has := modules[name]; has {
		panic(fmt.Sprintf("module %s registed twice", name))
	}
	modules[name] = mod
}

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range modules {
		names = append(names, name)
	}
	return names
}

// GetModuleMap returns the module map that includes all modules
// for the given module names.
func GetModuleMap(names ...string) *tengo.ModuleMap {
	mp := tengo.NewModuleMap()
	for _, name := range names {
		if mod := modules[name]; mod != nil {
			mp.AddBuiltinModule(name, mod)
		}
	}
	return mp
}

// WrapError wraps native go error into tengo.Error
func WrapError(err error) tengo.Object {
	if err == nil {
		return tengo.TrueValue
	}
	return &tengo.Error{Value: &tengo.String{Value: err.Error()}}
}

// ExtendObj extends tengo object to be able to use "extended" methods.
// See eobjects.go
func ExtendObj(o tengo.Object) (tengo.Object, error) {
	switch o := o.(type) {
	case *tengo.String:
		return &String{*o}, nil
	case *tengo.Array:
		return &Array{*o}, nil
	case *String:
		return o, nil
	case *Array:
		return o, nil
	default:
		return nil, ErrNotExtendable
	}
}
