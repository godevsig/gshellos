package gshellos

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/d5/tengo/v2"
)

func TestToObject(t *testing.T) {
	testb := true
	var testi uint32 = 88
	testf := 33.33
	var itf interface{}
	itf = testf

	type student struct {
		Name    string
		Age     int
		Scores  map[string]float32
		Friends []student
	}

	empty := struct {
		A string
		B int
		C float64
		D []int
		E []int64
		F []float64
		G []tengo.Object
		H []string
		I []byte
		J []interface{}
		K tengo.CallableFunc
		L error
		M map[string]int
		N map[string]int64
		O map[string]float64
		P map[string]string
		Q map[string]interface{}
		R map[string]tengo.Object
		S *int
	}{}

	cases := []struct {
		v    interface{}
		want string
	}{
		{&tengo.Int{Value: 123}, `&tengo.Int{ObjectImpl:tengo.ObjectImpl{}, Value:123}`},
		{nil, `&tengo.Undefined{ObjectImpl:tengo.ObjectImpl{}}`},
		{1, `&tengo.Int{ObjectImpl:tengo.ObjectImpl{}, Value:1}`},
		{"hello world", `&tengo.String{ObjectImpl:tengo.ObjectImpl{}, Value:"hello world", runeStr:[]int32(nil)}`},
		{99.99, `&tengo.Float{ObjectImpl:tengo.ObjectImpl{}, Value:99.99}`},
		{false, `&tengo.Bool{ObjectImpl:tengo.ObjectImpl{}, value:false}`},
		{'@', `&tengo.Char{ObjectImpl:tengo.ObjectImpl{}, Value:64}`},
		{byte(56), `&tengo.Char{ObjectImpl:tengo.ObjectImpl{}, Value:56}`},
		{[]byte("567"), `&tengo.Bytes{ObjectImpl:tengo.ObjectImpl{}, Value:[]uint8{0x35, 0x36, 0x37}}`},
		{errors.New("err"), `&tengo.Error{ObjectImpl:tengo.ObjectImpl{}, Value:\(\*tengo.String\)\(0x[0-9a-f]+\)}`},
		{map[string]int{"zhangsan": 30, "lisi": 35},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((zhangsan)|(lisi))":\(\*tengo.Int\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{map[string]int64{"zhangsan": 30, "lisi": 35},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((zhangsan)|(lisi))":\(\*tengo.Int\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{map[string]string{"zhangsan": "teacher", "lisi": "student"},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((zhangsan)|(lisi))":\(\*tengo.String\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{map[string]interface{}{"zhangsan": 30, "lisi": "student"},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((zhangsan)|(lisi))":\(\*tengo.((String)|(Int))\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{map[string]float64{"zhangsan": 30.1, "lisi": 35.2},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((zhangsan)|(lisi))":\(\*tengo.Float\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{[2]int{11, 13},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.Int\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{[]int{101, 103, 105},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.Int\)\(0x[0-9a-f]+\),? ?){3}}}$`},
		{[]int64{101, 103, 105},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.Int\)\(0x[0-9a-f]+\),? ?){3}}}$`},
		{[]float64{101.1, 103.1, 105.1},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.Float\)\(0x[0-9a-f]+\),? ?){3}}}$`},
		{[]string{"ni", "hao", "ma"},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.String\)\(0x[0-9a-f]+\),? ?){3}}}$`},
		{[]interface{}{"ni", "hao", 123},
			`^&tengo.Array{ObjectImpl:tengo.ObjectImpl{}, Value:\[\]tengo.Object{(\(\*tengo.((String)|(Int))\)\(0x[0-9a-f]+\),? ?){3}}}$`},
		{time.Now(), `^&tengo.Time{ObjectImpl:tengo.ObjectImpl{}, Value:time.Time{.*}}$`},
		{&testb, `&tengo.Bool{ObjectImpl:tengo.ObjectImpl{}, value:true}`},
		{int16(55), `&tengo.Int{ObjectImpl:tengo.ObjectImpl{}, Value:55}`},
		{&testi, `&tengo.Int{ObjectImpl:tengo.ObjectImpl{}, Value:88}`},
		{&testf, `&tengo.Float{ObjectImpl:tengo.ObjectImpl{}, Value:33.33}`},
		{itf, `&tengo.Float{ObjectImpl:tengo.ObjectImpl{}, Value:33.33}`},
		{student{"lisi", 20, map[string]float32{"yuwen": 86.5, "shuxue": 83.1}, []student{{Name: "zhangsan"}, {Name: "wangwu"}}},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((age)|(friends)|(name)|(scores))":\(\*tengo.((Int)|Array|String|Map)\)\(0x[0-9a-f]+\),? ?){4}}}$`},
		{map[string]student{"zhangsan": {Name: "zhangsan"}, "lisi": {Name: "lisi"}},
			`^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{("((lisi)|(zhangsan))":\(\*tengo.Map\)\(0x[0-9a-f]+\),? ?){2}}}$`},
		{empty, `^&tengo.Map{ObjectImpl:tengo.ObjectImpl{}, Value:map\[string\]tengo.Object{"a":\(\*tengo.String\)\(0x[0-9a-f]+\), "b":\(\*tengo.Int\)\(0x[0-9a-f]+\), "c":\(\*tengo.Float\)\(0x[0-9a-f]+\), ("[d-s]{1}":\(\*tengo.Undefined\)\(0x[0-9a-f]+\),? ?){16}}}$`},
	}

	for _, c := range cases {
		if len(c.want) == 0 {
			t.Error("empty want")
		}
		obj, err := ToObject(c.v)
		if err != nil {
			t.Error(err)
			continue
		}
		got := fmt.Sprintf("%#v", obj)
		//t.Logf("%v\n", obj)
		if got == c.want {
			continue
		}
		re := regexp.MustCompile(c.want)
		if !re.MatchString(got) {
			t.Errorf("want: %s, got: %s", c.want, got)
		}
	}

	_, err := ToObject([]complex64{complex(1, -2), complex(1.0, -1.4)})
	if err != ErrInvalidType {
		t.Error("complex supported?")
	}

	_, err = ToObject(map[string]interface{}{"a": complex(1, -2), "b": complex(1.0, -1.4)})
	if err != ErrInvalidType {
		t.Error("complex supported?")
	}
}

func TestFromObject(t *testing.T) {
	obj, _ := ToObject(55)
	emptyCases := []interface{}{
		(*int)(nil),
		(*int64)(nil),
		(*string)(nil),
		(*float64)(nil),
		(*bool)(nil),
		(*rune)(nil),
		(*[]byte)(nil),
		(*time.Time)(nil),
		(*[]int)(nil),
		(*[]int64)(nil),
		(*[]float64)(nil),
		(*[]string)(nil),
		(*map[string]int)(nil),
		(*map[string]int64)(nil),
		(*map[string]float64)(nil),
		(*map[string]string)(nil),
		(*tengo.Object)(nil),
		(*tengo.CallableFunc)(nil),
		(*int32)(nil),
		nil,
	}

	for _, c := range emptyCases {
		err := FromObject(c, obj)
		if err != ErrInvalidPtr {
			t.Fatal("empty ptr error expected")
		}
	}

	var got tengo.Object
	err := FromObject(&got, obj)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, obj) {
		t.Errorf("want: %#v, got: %#v", obj, got)
	}

	testf := func(args ...tengo.Object) (tengo.Object, error) {
		return nil, nil
	}
	var gotf func(args ...tengo.Object) (tengo.Object, error)
	fobj, err := ToObject(testf)
	if err != nil {
		t.Error(err)
	}
	err = FromObject(&gotf, fobj)
	if err != nil {
		t.Error(err)
	}
	var itf interface{} = gotf
	gotstring := fmt.Sprintf("%#v", itf)
	wantstring := `(func(...tengo.Object) (tengo.Object, error))`
	if !strings.Contains(gotstring, wantstring) {
		t.Errorf("want: %s, got: %s", wantstring, gotstring)
	}
	err = FromObject(&gotf, obj)
	if !errors.As(err, &ErrNotConvertibleType{}) {
		t.Error(err)
	}

	type student struct {
		Name       string
		Age        int
		Scores     map[string]float32
		Classmates []student
		Deskmate   *student
		Friends    map[string]*student
	}

	studentA := student{
		"lisi",
		20,
		map[string]float32{"yuwen": 86.5, "shuxue": 83.1},
		[]student{{Name: "zhangsan"}, {Name: "wangwu"}},
		nil,
		nil,
	}

	studentB := student{
		"zhangsan",
		21,
		map[string]float32{"yuwen": 78.5, "shuxue": 96.1},
		[]student{{Name: "lisi"}, {Name: "wangwu"}},
		&studentA,
		map[string]*student{"si": &studentA},
	}

	cases := []interface{}{
		"hello world",
		55,
		int64(33),
		55.77,
		true,
		'U',
		[]byte{1, 2, 3, 4, 5},
		time.Now(),
		[]int{22, 33, 44},
		[]int64{22, 33, 44},
		[]float64{22.1, 33.2, 44.9},
		[]string{"ni", "hao", "ma"},
		map[string]int{"A": 1, "b": 15},
		map[string]int64{"A": 1, "b": 15},
		map[string]float64{"a": 1.54, "U": 3.14},
		map[string]string{"a": "12345", "U": "hello world"},
		int16(12),
		uint16(12),
		float32(1.2345),
		studentB,
		studentA,
	}

	for _, c := range cases {
		t.Logf("c: %#v", c)
		obj, err := ToObject(c)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("obj: %#v", obj)

		ptr := reflect.New(reflect.TypeOf(c))
		err = FromObject(ptr.Interface(), obj)
		if err != nil {
			t.Error(err)
			continue
		}
		v := ptr.Elem().Interface()
		t.Logf("v: %#v", v)
		if !reflect.DeepEqual(c, v) {
			t.Errorf("want: %#v, got: %#v", c, v)
		}
	}
	//t.Error("err")
}
