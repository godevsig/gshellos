package gshellos

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/d5/tengo/v2"
)

// Object is an alias of tengo.Object
type Object = tengo.Object

var (
	// TrueValue represents a true value.
	TrueValue = tengo.TrueValue
	// FalseValue represents a false value.
	FalseValue = tengo.FalseValue
	// UndefinedValue represents an undefined value.
	UndefinedValue = tengo.UndefinedValue
)

var (
	// ErrWrongNumArguments represents a wrong number of arguments error.
	ErrWrongNumArguments = tengo.ErrWrongNumArguments

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
	// ErrBrokenGre is an error where the specified gre has problem to run.
	ErrBrokenGre = errors.New("broken gre")
)

// modules are extension modules managed by this package.
var (
	modules = map[string]map[string]tengo.Object{}
)

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

type endlessReader struct {
	r io.Reader
}

func (er endlessReader) Read(p []byte) (n int, err error) {
	for i := 0; i < 30; i++ {
		n, err = er.r.Read(p)
		if err != io.EOF {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	p[n] = 0 // fake read
	return n + 1, nil
}

type writerStat struct {
	w   io.Writer
	eof bool
}
type mWriter struct {
	writers []*writerStat
	eofNum  int
}

func (mw *mWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		if w.eof {
			continue
		}
		if _, err := w.w.Write(p); err != nil {
			w.eof = true
			mw.eofNum++
		}
	}
	if mw.eofNum == len(mw.writers) {
		return 0, io.EOF
	}
	return len(p), nil
}

// multiWriter is like io.MultiWriter but only stops if all writers are EOF.
func multiWriter(writers ...io.Writer) io.Writer {
	mw := &mWriter{}
	for _, w := range writers {
		mw.writers = append(mw.writers, &writerStat{w: w})
	}
	return mw
}

type null struct{}

func (null) Close() error                  { return nil }
func (null) Write(buf []byte) (int, error) { return len(buf), nil }
func (null) Read(buf []byte) (int, error)  { return 0, io.EOF }
