package jsonpath

import (
	"reflect"
	"strconv"
)

type subscript struct {
	name *string
}

var _ rule = &subscript{}

func (s *subscript) execute(in []value) (out []value) {
	for _, i := range in {
		v := i.Get()
		if !v.IsValid() {
			continue
		}
		v = skipPointers(v)

		switch v.Kind() {
		case reflect.Map:
			if s.name != nil {
				out = append(out, mapval{m: v, k: reflect.ValueOf(*s.name)})
			} else {
				for _, k := range v.MapKeys() {
					out = append(out, mapval{m: v, k: k})
				}
			}

		case reflect.Slice:
			if s.name != nil {
				i, err := strconv.Atoi(*s.name)
				if err != nil || i < 0 || i >= v.Len() {
					continue
				}
				out = append(out, val{v: v.Index(i)})
			} else {
				for i := 0; i < v.Len(); i++ {
					out = append(out, val{v: v.Index(i)})
				}
			}
		}
	}

	return
}
