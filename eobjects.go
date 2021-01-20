package gshellos

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/d5/tengo/v2"
)

type tString = tengo.String // to avoid field name confliction with method name

// String represents a string value.
type String struct {
	tString
}

// TypeName returns the name of the type.
func (o *String) TypeName() string {
	return "estring"
}

// String returns raw string
func (o *String) String() string {
	return o.Value
}

// refs:
// https://github.com/d5/tengo/blob/master/docs/stdlib-text.md
// https://www.abs-lang.org/types/string
// https://www.w3schools.com/python/python_ref_string.asp
var stringMethods = map[string]func(o *String) *tengo.UserFunction{
	// return array splitted by seperators
	"split": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "split", Value: o.split} },
	// return true if substr found
	"contains": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "contains", Value: o.contains} },
	/*
		// strips whitespaces
		"strip": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "strip", Value: o.strip} },
		// to upper
		"upper": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "upper", Value: o.upper} },
		// to lower
		"lower": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "lower", Value: o.lower} },
		// returns lowest index if substr if found, -1 otherwise
		"find": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "find", Value: o.find} },
		// returns new string by replacing some parts of the original string
		"replace": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "replace", Value: o.replace} },
		// returns array splitted by new line seperators
		"lines": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "lines", Value: o.lines} },
		// returns new string by repeating the original one certain times
		"repeat": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "repeat", Value: o.repeat} },
		// returns substr between an index range
		"substr": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "substr", Value: o.substr} },
		// returns new string by removing all substr
		"remove": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "remove", Value: o.remove} },
		// appends another string
		"append": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "append", Value: o.append} },
		// returns number of occurrance of the substr
		"count": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "count", Value: o.count} },
		// returns true if the string starts with substr
		"startswith": func(o *String) *tengo.UserFunction {
			return &tengo.UserFunction{Name: "startswith", Value: o.startswith}
		},
		// returns true if the string ends with substr
		"endswith": func(o *String) *tengo.UserFunction { return &tengo.UserFunction{Name: "endswith", Value: o.endswith} },
	*/
}

// IndexGet returns a character at a given index, or a builtin function of the given method.
func (o *String) IndexGet(index tengo.Object) (tengo.Object, error) {
	res, err := o.tString.IndexGet(index)
	if !errors.Is(err, tengo.ErrInvalidIndexType) {
		return res, err
	}
	idx, ok := index.(*tengo.String)
	if !ok {
		return nil, tengo.ErrInvalidIndexType
	}

	if method, ok := stringMethods[idx.Value]; ok {
		return method(o), nil
	}

	return nil, ErrUndefinedMethod
}

// splits the string into array.
//  no arg: split by whitespaces
//  1 arg: split by the input sep
//  >1 args: each arg is a sep, return array as if it was proccessed one by one
func (o *String) split(args ...tengo.Object) (tengo.Object, error) {
	var ss []string
	nargs := len(args)
	if nargs == 0 {
		ss = strings.Fields(o.Value) // split by whitespaces
	} else if nargs == 1 {
		sep, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{
				Name:     "arg 0",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		ss = strings.Split(o.Value, sep)
	} else {
		seps := make([]string, nargs)
		for i, a := range args {
			sep, ok := tengo.ToString(a)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{
					Name:     "arg " + strconv.FormatInt(int64(i), 10),
					Expected: "string(compatible)",
					Found:    a.TypeName(),
				}
			}
			seps[i] = sep
		}

		sort.Slice(seps, func(i, j int) bool { return len(seps[i]) > len(seps[j]) })
		s := o.Value
		tmp := `临时用⌘代替`
		for _, sep := range seps {
			s = strings.ReplaceAll(s, sep, tmp)
		}
		ss = strings.Split(s, tmp)
	}

	if len(ss) == 0 {
		return &Array{}, nil
	}

	arr := make([]tengo.Object, len(ss))
	for i, e := range ss {
		arr[i] = &tengo.String{Value: e}
	}
	return &Array{tArray{Value: arr}}, nil
}

// ToDo
func (o *String) contains(args ...tengo.Object) (tengo.Object, error) {
	return tengo.FalseValue, nil
}

type tArray = tengo.Array // to avoid field name confliction with method name

// Array represents an array of objects.
type Array struct {
	tArray
}

// TypeName returns the name of the type.
func (o *Array) TypeName() string {
	return "earray"
}

// refs:
// https://www.abs-lang.org/types/array
// https://www.w3schools.com/python/python_ref_list.asp
var arrayMethods = map[string]func(o *Array) *tengo.UserFunction{
	// returns a string after joined by seperator string
	"join": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "join", Value: o.join} },
	/*
		// inserts an element at the end of the array
		"push": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "push", Value: o.push} },
		// removes and returns the last element from the array
		"pop": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "pop", Value: o.pop} },
		// removes and returns the first element from the array
		"shift": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "shift", Value: o.shift} },
		// adds the elements of another array to the end of the current one
		"extend": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "extend", Value: o.extend} },
		// adds an element at the specified position
		"insert": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "insert", Value: o.insert} },
		// deltes an element at the specified position
		"delete": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "delete", Value: o.delete} },
		// returns a new array with every element x that returned true by fn(x)
		"filter": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "filter", Value: o.filter} },
		// returns a new array with fn(x) on each element x
		"each": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "each", Value: o.each} },
		// recursively flattens an array until no element is an array
		"flatten": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "flatten", Value: o.flatten} },
		// sorts the array. Only supported on homogeneous arrays of numbers or strings
		"sort": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "sort", Value: o.sort} },
		// returns the array with duplicate values removed. The values need not be sorted
		"unique": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "unique", Value: o.unique} },
		// removes the first occurrence of the element with the specified value
		"remove": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "remove", Value: o.remove} },
		// returns the position at the first occurrence of the specified value, -1 if not found
		"find": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "find", Value: o.find} },
		// returns the number of elements with the specified value
		"count": func(o *Array) *tengo.UserFunction { return &tengo.UserFunction{Name: "count", Value: o.count} },
	*/
}

// IndexGet returns an element at a given index, or a builtin function of the given method.
func (o *Array) IndexGet(index tengo.Object) (tengo.Object, error) {
	switch idx := index.(type) {
	case *tengo.Int:
		idxVal := int(idx.Value)
		if idxVal < 0 {
			idxVal = len(o.Value) + idxVal
		}

		if idxVal < 0 || idxVal >= len(o.Value) {
			return tengo.UndefinedValue, nil
		}

		res := o.Value[idxVal]
		if extd, err := ExtendObj(res); err == nil {
			res = extd
		}
		return res, nil

	case *tengo.String:
		if method, ok := arrayMethods[idx.Value]; ok {
			return method(o), nil
		}
		return nil, ErrUndefinedMethod
	default:
		return nil, tengo.ErrInvalidIndexType
	}
}

// join array into string
func (o *Array) join(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}

	sep, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{
			Name:     "arg 0",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}

	var slen int
	var ss []string
	for i, v := range o.Value {
		s, ok := tengo.ToString(v)
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{
				Name:     "array index " + strconv.FormatInt(int64(i), 10),
				Expected: "string(compatible)",
				Found:    v.TypeName(),
			}
		}
		slen += len(s)
		ss = append(ss, s)
	}

	// make sure output length does not exceed the limit
	if slen+len(sep)*(len(ss)-1) > tengo.MaxStringLen {
		return nil, tengo.ErrStringLimit
	}

	return &String{tString{Value: strings.Join(ss, sep)}}, nil
}

// UserFunction represents a callable function.
type UserFunction struct {
	tengo.ObjectImpl
	//Name      string
	Value     tengo.CallableFunc
	Signature string
	Usage     string
	Example   string
}

// TypeName returns the name of the type.
func (o *UserFunction) TypeName() string {
	i := strings.Index(o.Signature, "(")
	if i == -1 {
		i = 0
	}
	return "efunction: " + o.Signature[:i]
}

func (o *UserFunction) String() string {
	return "<efunction>"
}

// Help returns the help message
func (o *UserFunction) Help() string {
	var help strings.Builder
	help.WriteString(o.String())
	help.WriteString(":\n")
	help.WriteString("Signature\n")
	for _, s := range strings.Split(o.Signature, "\n") {
		help.WriteString("    ")
		help.WriteString(strings.TrimSpace(s))
		help.WriteString("\n")
	}

	help.WriteString("Usage\n")
	for _, s := range strings.Split(o.Usage, "\n") {
		help.WriteString("    ")
		help.WriteString(strings.TrimSpace(s))
		help.WriteString("\n")
	}

	help.WriteString("Example\n")
	for _, s := range strings.Split(o.Example, "\n") {
		help.WriteString("    ")
		help.WriteString(strings.TrimSpace(s))
		help.WriteString("\n")
	}

	return help.String()
}

// Copy returns a copy of the type.
func (o *UserFunction) Copy() tengo.Object {
	cp := UserFunction{}
	cp = *o
	return &cp
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *UserFunction) Equals(_ tengo.Object) bool {
	return false
}

// Call invokes a user function.
func (o *UserFunction) Call(args ...tengo.Object) (tengo.Object, error) {
	return o.Value(args...)
}

// CanCall returns whether the Object can be Called.
func (o *UserFunction) CanCall() bool {
	return true
}
