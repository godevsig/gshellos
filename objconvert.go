package gshellos

import (
	"fmt"
	"reflect"
	"time"

	"github.com/d5/tengo/v2"
)

func firstLetterLower(s string) string {
	c := s[0]
	if c <= 'Z' {
		return string(c+32) + s[1:]
	}
	return s
}

// ToObject traverses the value v recursively and converts the value to tengo object.
//  Pointer values encode as the value pointed to.
//  A nil pointer/interface/slice/map encodes as the tengo.UndefinedValue value.
//  Struct values encode as tengo map. Only exported field can be encoded with filed names as map keys but with its first letter turned into lower case.
//    e.g. struct{Field1: 123, AnotherField: 456} will be converted to tengo map{field: 123, anotherField: 456}
//  int, string, float, bool, Time.time, error encodes as their corresponding tengo object.
//  slices encode as tengo Array, maps with key as string encode as tengo Map, returns ErrInvalidType if key type in map is not string.
// Returns ErrInvalidType on unsupported value type.
// Note as ToObject follows pointers, be careful with cyclic pointer references which results in infinite loop.
func ToObject(v interface{}) (tengo.Object, error) {
	// fast path
	switch v := v.(type) {
	case nil:
		return tengo.UndefinedValue, nil
	case string:
		if len(v) > tengo.MaxStringLen {
			return nil, tengo.ErrStringLimit
		}
		return &tengo.String{Value: v}, nil
	case int64:
		return &tengo.Int{Value: v}, nil
	case int:
		return &tengo.Int{Value: int64(v)}, nil
	case bool:
		if v {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	case rune:
		return &tengo.Char{Value: v}, nil
	case byte:
		return &tengo.Char{Value: rune(v)}, nil
	case float64:
		return &tengo.Float{Value: v}, nil
	case *UserFunction:
		return v, nil
	case *tengo.UserFunction:
		return v, nil
	case tengo.Object:
		return v, nil
	case tengo.CallableFunc:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		return &tengo.UserFunction{Value: v}, nil
	case []byte:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		if len(v) > tengo.MaxBytesLen {
			return nil, tengo.ErrBytesLimit
		}
		return &tengo.Bytes{Value: v}, nil
	case error:
		if v == nil {
			return tengo.TrueValue, nil
		}
		return &tengo.Error{Value: &tengo.String{Value: v.Error()}}, nil
	case map[string]tengo.Object:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		return &tengo.Map{Value: v}, nil
	case map[string]int:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		kv := make(map[string]tengo.Object, len(v))
		for vk, vv := range v {
			vo, err := ToObject(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &tengo.Map{Value: kv}, nil
	case map[string]int64:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		kv := make(map[string]tengo.Object, len(v))
		for vk, vv := range v {
			vo, err := ToObject(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &tengo.Map{Value: kv}, nil
	case map[string]float64:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		kv := make(map[string]tengo.Object, len(v))
		for vk, vv := range v {
			vo, err := ToObject(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &tengo.Map{Value: kv}, nil
	case map[string]string:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		kv := make(map[string]tengo.Object, len(v))
		for vk, vv := range v {
			vo, err := ToObject(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &tengo.Map{Value: kv}, nil
	case map[string]interface{}:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		kv := make(map[string]tengo.Object, len(v))
		for vk, vv := range v {
			vo, err := ToObject(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &tengo.Map{Value: kv}, nil
	case []tengo.Object:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		return &tengo.Array{Value: v}, nil
	case []int:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			vo, err := ToObject(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &tengo.Array{Value: arr}, nil
	case []int64:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			vo, err := ToObject(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &tengo.Array{Value: arr}, nil
	case []float64:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			vo, err := ToObject(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &tengo.Array{Value: arr}, nil
	case []string:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			vo, err := ToObject(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &tengo.Array{Value: arr}, nil
	case []interface{}:
		if v == nil {
			return tengo.UndefinedValue, nil
		}
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			vo, err := ToObject(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &tengo.Array{Value: arr}, nil
	case time.Time:
		return &tengo.Time{Value: v}, nil
	}

	// slow path
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		if rv.IsNil() {
			return tengo.UndefinedValue, nil
		}
	}

	rv = reflect.Indirect(rv)
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &tengo.Int{Value: rv.Int()}, nil
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &tengo.Int{Value: int64(rv.Uint())}, nil
	case reflect.Float32, reflect.Float64:
		return &tengo.Float{Value: rv.Float()}, nil
	case reflect.Array, reflect.Slice:
		arr := make([]tengo.Object, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			obj, err := ToObject(rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			arr[i] = obj
		}
		return &tengo.Array{Value: arr}, nil
	case reflect.Interface:
		obj, err := ToObject(rv.Elem().Interface())
		if err != nil {
			return nil, err
		}
		return obj, nil
	case reflect.Map:
		kv := make(map[string]tengo.Object, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				return nil, ErrInvalidType
			}
			v := iter.Value()
			obj, err := ToObject(v.Interface())
			if err != nil {
				return nil, err
			}
			kv[k.String()] = obj
		}
		return &tengo.Map{Value: kv}, nil
	case reflect.Struct:
		kv := make(map[string]tengo.Object, rv.NumField())
		typ := rv.Type()

		for i := 0; i < rv.NumField(); i++ {
			obj, err := ToObject(rv.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			kv[firstLetterLower(typ.Field(i).Name)] = obj
		}
		return &tengo.Map{Value: kv}, nil
	}

	return nil, ErrInvalidType
}

// MustToObject is like ToObject, but it panics if errors happen.
func MustToObject(v interface{}) tengo.Object {
	obj, err := ToObject(v)
	if err != nil {
		panic(err)
	}
	return obj
}

// ErrNotConvertibleType is an error when failed to convert tengo object.
type ErrNotConvertibleType struct {
	Expected string
	Found    interface{}
}

func (e ErrNotConvertibleType) Error() string {
	return fmt.Sprintf("invalid type, expected: %s, found: %v",
		e.Expected, e.Found)
}

func errNotConvertible(expected string, found interface{}) error {
	return ErrNotConvertibleType{expected, found}
}

// FromObject parses the tengo object and stores the result in the value pointed to by v.
// FromObject uses the inverse of the encodings that ToObject uses, allocating maps, slices, and pointers as necessary.
//
// FromObject converts tengo Map object into a struct by map look up with field names as keys.
// Filed name and tengo map key are matched in a way that the first letter is insensitive.
//   e.g. both tengo map{name: "san"} and map{Name: "san"} can be converted to struct{Name: "san"}
//  If v is nil or not a pointer, ObjectToValue returns an ErrInvalidPtr error.
//  If o is already a tengo object, it is copied to the value that v points to.
//  If v represents a *tengo.CallableFunc, and o is a tengo UserFunction object, the CallableFunc f will be copied to where v points.
// Returns ErrNotConvertibleType if o can not be converted to v, e.g. you are trying to get a map vale from tengo Array object.
// Not supported value types:
//  interface, chan, complex, func
//  In particular, interface error is not convertible.
func FromObject(v interface{}, o tengo.Object) error {
	if o == tengo.UndefinedValue {
		return nil // ignore undefined value
	}

	// fast path
	switch ptr := v.(type) {
	case *int:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToInt(o); ok {
			*ptr = v
			return nil
		}
	case *int64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToInt64(o); ok {
			*ptr = v
			return nil
		}
	case *string:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToString(o); ok {
			*ptr = v
			return nil
		}
	case *float64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToFloat64(o); ok {
			*ptr = v
			return nil
		}
	case *bool:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToBool(o); ok {
			*ptr = v
			return nil
		}
	case *rune:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToRune(o); ok {
			*ptr = v
			return nil
		}
	case *[]byte:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToByteSlice(o); ok {
			*ptr = v
			return nil
		}
	case *time.Time:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if v, ok := tengo.ToTime(o); ok {
			*ptr = v
			return nil
		}
	case *[]int:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toA := func(objArray []tengo.Object) bool {
			array := make([]int, len(objArray))
			for i, o := range objArray {
				v, ok := tengo.ToInt(o)
				if !ok {
					return false
				}
				array[i] = v
			}
			*ptr = array
			return true
		}
		switch o := o.(type) {
		case *tengo.Array:
			if toA(o.Value) {
				return nil
			}
		case *tengo.ImmutableArray:
			if toA(o.Value) {
				return nil
			}
		}
	case *[]int64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toA := func(objArray []tengo.Object) bool {
			array := make([]int64, len(objArray))
			for i, o := range objArray {
				v, ok := tengo.ToInt64(o)
				if !ok {
					return false
				}
				array[i] = v
			}
			*ptr = array
			return true
		}
		switch o := o.(type) {
		case *tengo.Array:
			if toA(o.Value) {
				return nil
			}
		case *tengo.ImmutableArray:
			if toA(o.Value) {
				return nil
			}
		}
	case *[]float64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toA := func(objArray []tengo.Object) bool {
			array := make([]float64, len(objArray))
			for i, o := range objArray {
				v, ok := tengo.ToFloat64(o)
				if !ok {
					return false
				}
				array[i] = v
			}
			*ptr = array
			return true
		}
		switch o := o.(type) {
		case *tengo.Array:
			if toA(o.Value) {
				return nil
			}
		case *tengo.ImmutableArray:
			if toA(o.Value) {
				return nil
			}
		}
	case *[]string:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toA := func(objArray []tengo.Object) bool {
			array := make([]string, len(objArray))
			for i, o := range objArray {
				v, ok := tengo.ToString(o)
				if !ok {
					return false
				}
				array[i] = v
			}
			*ptr = array
			return true
		}
		switch o := o.(type) {
		case *tengo.Array:
			if toA(o.Value) {
				return nil
			}
		case *tengo.ImmutableArray:
			if toA(o.Value) {
				return nil
			}
		}
	case *map[string]int:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toM := func(objMap map[string]tengo.Object) bool {
			mp := make(map[string]int, len(objMap))
			for k, o := range objMap {
				v, ok := tengo.ToInt(o)
				if !ok {
					return false
				}
				mp[k] = v
			}
			*ptr = mp
			return true
		}
		switch o := o.(type) {
		case *tengo.Map:
			if toM(o.Value) {
				return nil
			}
		case *tengo.ImmutableMap:
			if toM(o.Value) {
				return nil
			}
		}
	case *map[string]int64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toM := func(objMap map[string]tengo.Object) bool {
			mp := make(map[string]int64, len(objMap))
			for k, o := range objMap {
				v, ok := tengo.ToInt64(o)
				if !ok {
					return false
				}
				mp[k] = v
			}
			*ptr = mp
			return true
		}
		switch o := o.(type) {
		case *tengo.Map:
			if toM(o.Value) {
				return nil
			}
		case *tengo.ImmutableMap:
			if toM(o.Value) {
				return nil
			}
		}
	case *map[string]float64:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toM := func(objMap map[string]tengo.Object) bool {
			mp := make(map[string]float64, len(objMap))
			for k, o := range objMap {
				v, ok := tengo.ToFloat64(o)
				if !ok {
					return false
				}
				mp[k] = v
			}
			*ptr = mp
			return true
		}
		switch o := o.(type) {
		case *tengo.Map:
			if toM(o.Value) {
				return nil
			}
		case *tengo.ImmutableMap:
			if toM(o.Value) {
				return nil
			}
		}
	case *map[string]string:
		if ptr == nil {
			return ErrInvalidPtr
		}
		toM := func(objMap map[string]tengo.Object) bool {
			mp := make(map[string]string, len(objMap))
			for k, o := range objMap {
				v, ok := tengo.ToString(o)
				if !ok {
					return false
				}
				mp[k] = v
			}
			*ptr = mp
			return true
		}
		switch o := o.(type) {
		case *tengo.Map:
			if toM(o.Value) {
				return nil
			}
		case *tengo.ImmutableMap:
			if toM(o.Value) {
				return nil
			}
		}
	case *tengo.Object:
		if ptr == nil {
			return ErrInvalidPtr
		}
		*ptr = o
		return nil
	case *tengo.CallableFunc:
		if ptr == nil {
			return ErrInvalidPtr
		}
		if f, ok := o.(*tengo.UserFunction); ok {
			*ptr = f.Value
			return nil
		}
	default:
		// slow path
		rptr := reflect.ValueOf(v)
		if rptr.Kind() != reflect.Ptr || rptr.IsNil() {
			return ErrInvalidPtr
		}
		rv := rptr.Elem()
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v, ok := tengo.ToInt64(o); ok {
				rv.SetInt(v)
				return nil
			}
		case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if v, ok := tengo.ToInt64(o); ok {
				rv.SetUint(uint64(v))
				return nil
			}
		case reflect.Float32, reflect.Float64:
			if v, ok := tengo.ToFloat64(o); ok {
				rv.SetFloat(v)
				return nil
			}
		case reflect.Ptr:
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			if err := FromObject(rv.Interface(), o); err == nil {
				return nil
			}
		case reflect.Array, reflect.Slice:
			toA := func(objArray []tengo.Object) bool {
				array := reflect.MakeSlice(rv.Type(), len(objArray), len(objArray))
				for i, o := range objArray {
					if o == tengo.UndefinedValue {
						continue
					}
					elem := array.Index(i)
					if err := FromObject(elem.Addr().Interface(), o); err != nil {
						return false
					}
				}
				rv.Set(array)
				return true
			}

			switch o := o.(type) {
			case *tengo.Array:
				if toA(o.Value) {
					return nil
				}
			case *tengo.ImmutableArray:
				if toA(o.Value) {
					return nil
				}
			}
		case reflect.Map:
			toM := func(objMap map[string]tengo.Object) bool {
				typ := rv.Type()
				if typ.Key().Kind() != reflect.String {
					return false
				}
				mp := reflect.MakeMapWithSize(typ, len(objMap))
				elemPtr := reflect.New(typ.Elem())
				for k, o := range objMap {
					if o == tengo.UndefinedValue {
						continue
					}
					if err := FromObject(elemPtr.Interface(), o); err != nil {
						return false
					}
					mp.SetMapIndex(reflect.ValueOf(k), elemPtr.Elem())
				}
				rv.Set(mp)
				return true
			}

			switch o := o.(type) {
			case *tengo.Map:
				if toM(o.Value) {
					return nil
				}
			case *tengo.ImmutableMap:
				if toM(o.Value) {
					return nil
				}
			}
		case reflect.Struct:
			toStruct := func(objMap map[string]tengo.Object) bool {
				typ := rv.Type()
				for i := 0; i < rv.NumField(); i++ {
					fieldName := typ.Field(i).Name
					obj, ok := objMap[firstLetterLower(fieldName)]
					if !ok {
						obj, ok = objMap[fieldName]
					}
					if ok {
						if obj == tengo.UndefinedValue {
							continue
						}
						field := rv.Field(i)
						if err := FromObject(field.Addr().Interface(), obj); err != nil {
							return false
						}
					}
				}
				return true
			}
			switch o := o.(type) {
			case *tengo.Map:
				if toStruct(o.Value) {
					return nil
				}
			case *tengo.ImmutableMap:
				if toStruct(o.Value) {
					return nil
				}
			}
		}
	}
	return errNotConvertible(reflect.ValueOf(v).Elem().String(), o)
}
