package jsonpath

import "reflect"

type value interface {
	Get() reflect.Value
	Set(reflect.Value)
	Delete()
}

type mapval struct {
	m, k reflect.Value
}

func (v mapval) Get() reflect.Value  { return v.m.MapIndex(v.k) }
func (v mapval) Set(x reflect.Value) { v.m.SetMapIndex(v.k, x) }
func (v mapval) Delete()             { v.m.SetMapIndex(v.k, reflect.Value{}) }

var _ value = mapval{}

type sliceval struct {
	s reflect.Value
	i int
}

type val struct {
	v reflect.Value
}

func (v val) Get() reflect.Value  { return reflect.Value(v.v) }
func (v val) Set(x reflect.Value) { v.v.Set(x) }
func (v val) Delete()             { panic("not implemented") }

var _ value = val{}
