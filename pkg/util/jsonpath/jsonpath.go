package jsonpath

//go:generate go get golang.org/x/tools/cmd/goyacc
//go:generate goyacc -l -o parser.go -v /dev/null parser.y

import (
	"bufio"
	"bytes"
	"reflect"
)

type Path interface {
	Delete(interface{})
	DeleteIfMatch(interface{}, interface{})
	Get(interface{}) []interface{}
	MustGetObject(interface{}) map[string]interface{}
	MustGetSlice(interface{}) []interface{}
	MustGetString(interface{}) string
	MustGetStrings(interface{}) []string
	Set(interface{}, interface{})
}

func Compile(s string) (Path, error) {
	yyErrorVerbose = true

	l := &lexer{r: bufio.NewReader(bytes.NewBufferString(s))}
	if yyParse(l) != 0 {
		return nil, l.err
	}

	return l.out.(Path), nil
}

func MustCompile(s string) Path {
	p, err := Compile(s)
	if err != nil {
		panic(err)
	}

	return p
}

type rule interface {
	execute([]value) []value
}

type rules []rule

var _ Path = rules{}

func (r rules) Get(i interface{}) (os []interface{}) {
	is := []value{val{v: reflect.ValueOf(i)}}
	for _, rule := range r {
		is = rule.execute(is)
	}

	for _, i := range is {
		v := i.Get()
		if v.IsValid() {
			os = append(os, v.Interface())
		}
	}

	return
}

func (r rules) MustGetObject(i interface{}) map[string]interface{} {
	if len(r.Get(i)) == 0 {
		return nil
	}
	return r.Get(i)[0].(map[string]interface{})
}

func (r rules) MustGetSlice(i interface{}) (ss []interface{}) {
	for _, s := range r.Get(i)[0].([]interface{}) {
		ss = append(ss, s)
	}
	return
}

func (r rules) MustGetString(i interface{}) string {
	return r.Get(i)[0].(string)
}

func (r rules) MustGetStrings(i interface{}) (ss []string) {
	for _, s := range r.Get(i) {
		ss = append(ss, s.(string))
	}
	return
}

func (r rules) Set(i interface{}, x interface{}) {
	is := []value{val{v: reflect.ValueOf(i)}}
	for _, rule := range r {
		is = rule.execute(is)
	}

	for _, i := range is {
		i.Set(reflect.ValueOf(x))
	}

	return
}

func (r rules) Delete(i interface{}) {
	is := []value{val{v: reflect.ValueOf(i)}}
	for _, rule := range r {
		is = rule.execute(is)
	}

	for _, i := range is {
		if i.Get().IsValid() {
			i.Delete()
		}
	}

	return
}

func (r rules) DeleteIfMatch(i interface{}, j interface{}) {
	is := []value{val{v: reflect.ValueOf(i)}}
	for _, rule := range r {
		is = rule.execute(is)
	}

	for _, i := range is {
		if i.Get().IsValid() && reflect.DeepEqual(i.Get().Interface(), j) {
			i.Delete()
		}
	}

	return
}

func skipPointers(i reflect.Value) reflect.Value {
	for i.Kind() == reflect.Interface || i.Kind() == reflect.Ptr {
		i = i.Elem()
	}
	return i
}
