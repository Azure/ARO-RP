package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
)

func TestMissingFields(t *testing.T) {
	result := finder(reflect.ValueOf(OpenShiftCluster{}), make(map[string]bool))
	for k, v := range result {
		if !v {
			t.Errorf("structure %s missing MissingFields field", k)
		}
	}
}

func finder(v reflect.Value, found map[string]bool) map[string]bool {
	var finder func(reflect.Value, map[string]bool) map[string]bool

	finder = func(v reflect.Value, found map[string]bool) map[string]bool {
		// Drill down through pointers and interfaces to get a value we can print.
		for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
			v = v.Elem()
		}

		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				found = finder(v.Index(i), found)
			}
		case reflect.Struct:
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				if t.Field(i).Name == "MissingFields" {
					found[t.Name()] = true
				}
				found = finder(v.Field(i), found)
			}
			if _, ok := found[t.Name()]; !ok && t.Name() != "MissingFields" {
				found[t.Name()] = false
			}
		default:
		}
		return found
	}
	return finder(v, found)
}
