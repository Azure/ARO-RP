package jsonpath

import (
	"reflect"
)

type filter struct {
	r rules
	s string
}

var _ rule = &filter{}

func (f *filter) execute(is []value) (out []value) {
	for _, i := range is {
		v := i.Get()
		if !v.IsValid() {
			continue
		}
		v = skipPointers(v)

		switch v.Kind() {
		case reflect.Map:
			for _, k := range v.MapKeys() {
				res := f.r.Get(v.MapIndex(k).Interface())
				if len(res) == 1 && res[0].(string) == f.s {
					out = append(out, mapval{m: v, k: k})
				}
			}

		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				res := f.r.Get(v.Index(i).Interface())
				if len(res) == 1 && res[0].(string) == f.s {
					out = append(out, val{v: v.Index(i)})
				}
			}
		}
	}

	return
}
