package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	gofrsuuid "github.com/gofrs/uuid"
)

const (
	track2sdkPkgPathPrefix = "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager"
)

// MarshalJSON marshals the nested r.Resource ignoring any MarshalJSON() methods
// on its types.  It then merges remaining fields of r over the result
func (r *Resource) MarshalJSON() ([]byte, error) {
	// hack to handle newer track2 sdk which doesn't have json tags
	v := reflect.Indirect(reflect.ValueOf(r.Resource))

	if strings.HasPrefix(v.Type().PkgPath(), track2sdkPkgPathPrefix) {
		res, ok := r.Resource.(json.Marshaler)
		if !ok {
			return nil, fmt.Errorf("resource %s identified as track2 sdk struct not marshalable", v.Type().Name())
		}
		b, err := res.MarshalJSON()

		if err != nil {
			return b, err
		}
		dataMap := map[string]interface{}{}
		err = json.Unmarshal(b, &dataMap)
		if err != nil {
			return nil, err
		}

		dataMap["apiVersion"] = r.APIVersion
		if r.DependsOn != nil {
			dataMap["dependsOn"] = r.DependsOn
		}
		return json.Marshal(dataMap)
	}

	resource := reflect.ValueOf(shadowCopy(r.Resource))
	outer := reflect.ValueOf(*r)

	if resource.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Resource field must be a struct")
	}

	// Create slices of field types and values that combine `r.Resource` and
	// outer `*r` structs.  Fields from the outer struct `*r` override fields
	// from `r.Resource`.
	fields := make([]reflect.StructField, 0, resource.NumField()+outer.NumField())
	values := make([]reflect.Value, 0, resource.NumField()+outer.NumField())
	indexes := map[string]int{}

	// Copy fields and values from `r.Resource`
	for i := 0; i < resource.NumField(); i++ {
		field := resource.Type().Field(i)
		fields = append(fields, field)
		values = append(values, resource.Field(i))
		indexes[field.Name] = i
	}

	// Copy fields and values from `*r` and override if they already exist
	for i := 1; i < outer.NumField(); i++ {
		field := outer.Type().Field(i)

		if j, found := indexes[field.Name]; found {
			field.Type = emptyInterfaceType
			fields[j] = field
			if !outer.Field(i).IsZero() {
				values[j] = outer.Field(i)
			}
		} else {
			fields = append(fields, field)
			values = append(values, outer.Field(i))
			indexes[field.Name] = len(fields) - 1
		}
	}

	combined := reflect.New(reflect.StructOf(fields)).Elem()

	for i, v := range values {
		combined.Field(i).Set(v)
	}

	return json.Marshal(combined.Interface())
}

// UnmarshalJSON is not implemented
func (r *Resource) UnmarshalJSON(b []byte) error {
	return fmt.Errorf("not implemented")
}

var (
	stringType         = reflect.TypeOf("")
	emptyInterfaceType = reflect.ValueOf([]interface{}(nil)).Type().Elem()
)

// shadowCopy returns a copy of the input object wherein all the struct types
// have been replaced with dynamically created ones.  The idea is that the
// JSONMarshal() methods get dropped in the process and so the returned object
// marshals natively.  Type cycles are permitted, but (as in encoding/json)
// value cycles are not.  Golang reflect doesn't support dynamically creating
// named types; to get around this we go weakly typed
func shadowCopy(i interface{}) interface{} {
	return _shadowCopy(reflect.ValueOf(i)).Interface()
}

func _shadowCopy(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Array:
		var t reflect.Type
		if v.Type() == reflect.TypeOf(gofrsuuid.UUID{}) {
			// keep uuid.UUID - encoding/json will detect it and marshal it into
			// a string
			t = v.Type()
		} else {
			t = reflect.ArrayOf(v.Len(), emptyInterfaceType)
		}
		a := reflect.New(t).Elem()
		for i := 0; i < v.Len(); i++ {
			a.Index(i).Set(_shadowCopy(v.Index(i)))
		}
		return a

	case reflect.Interface, reflect.Ptr:
		t := emptyInterfaceType
		if v.IsNil() {
			return reflect.Zero(t)
		}
		i := reflect.New(t).Elem()
		i.Set(_shadowCopy(v.Elem()))
		return i

	case reflect.Map:
		// this is not fully generic but Go json marshaling requires
		// map[string]interface{}
		t := reflect.MapOf(stringType, emptyInterfaceType)
		if v.IsNil() {
			return reflect.Zero(t)
		}
		m := reflect.MakeMap(t)
		iter := v.MapRange()
		for iter.Next() {
			m.SetMapIndex(iter.Key(), _shadowCopy(iter.Value()))
		}
		return m

	case reflect.Slice:
		var t reflect.Type
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// keep []byte - encoding/json will detect it and marshal it into a
			// base64 encoded string
			t = v.Type()
		} else {
			t = reflect.SliceOf(emptyInterfaceType)
		}
		if v.IsNil() {
			return reflect.Zero(t)
		}
		s := reflect.MakeSlice(t, v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			s.Index(i).Set(_shadowCopy(v.Index(i)))
		}
		return s

	case reflect.Struct:
		fields := make([]reflect.StructField, 0, v.Type().NumField())
		values := make([]reflect.Value, 0, v.Type().NumField())
		for i := 0; i < v.Type().NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			f := _shadowCopy(v.Field(i))
			values = append(values, f)

			field := v.Type().Field(i)
			field.Type = emptyInterfaceType
			fields = append(fields, field)
		}
		t := reflect.StructOf(fields)

		s := reflect.New(t).Elem()
		for i, v := range values {
			if !v.IsZero() {
				s.Field(i).Set(v)
			}
		}
		return s

	default:
		return v
	}
}
