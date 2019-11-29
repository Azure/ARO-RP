package v20191231preview

import (
	"reflect"
)

func deepCopy(in, out reflect.Value) {
	switch in.Kind() {
	case reflect.Array:
		a := reflect.New(in.Type()).Elem()
		for i := 0; i < in.Len(); i++ {
			deepCopy(in.Index(i), a.Index(i))
		}
		out.Set(a)

	case reflect.Interface:
		if in.IsNil() {
			return
		}

		o := reflect.New(in.Elem().Type()).Elem()
		deepCopy(in.Elem(), o)
		out.Set(o)

	case reflect.Map:
		if in.IsNil() {
			return
		}

		m := reflect.MakeMap(in.Type())
		for _, k := range in.MapKeys() {
			v := in.MapIndex(k)
			newk := reflect.New(in.Type().Key()).Elem()
			newv := reflect.New(in.Type().Elem()).Elem()
			deepCopy(k, newk)
			deepCopy(v, newv)
			m.SetMapIndex(newk, newv)
		}
		out.Set(m)

	case reflect.Ptr:
		if in.IsNil() {
			return
		}

		p := reflect.New(in.Type().Elem())
		deepCopy(in.Elem(), p.Elem())
		out.Set(p)

	case reflect.Slice:
		if in.IsNil() {
			return
		}

		s := reflect.MakeSlice(in.Type(), in.Len(), in.Cap())
		for i := 0; i < in.Len(); i++ {
			deepCopy(in.Index(i), s.Index(i))
		}
		out.Set(s)

	case reflect.Struct:
		out.Set(in) // copies unexported fields (best effort)

		for i := 0; i < in.NumField(); i++ {
			if out.Field(i).CanSet() {
				deepCopy(in.Field(i), out.Field(i))
			}
		}

	default:
		out.Set(in)
	}
}

func (in *OpenShiftCluster) DeepCopy() (out *OpenShiftCluster) {
	deepCopy(reflect.ValueOf(in), reflect.ValueOf(&out).Elem())
	return
}
