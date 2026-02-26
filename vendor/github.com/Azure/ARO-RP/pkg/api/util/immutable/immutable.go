package immutable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"strings"
)

type ValidationError struct {
	Target  string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validate returns nil if v and w are identical, bar any differences on any
// struct fields explicitly tagged `mutable:"true"`.  Otherwise it returns a
// CloudError indicating the first difference it finds
func Validate(path string, v, w interface{}) error {
	return validate(path, reflect.ValueOf(v), reflect.ValueOf(w), false)
}

func validate(path string, v, w reflect.Value, ignoreCase bool) error {
	if v.Type() != w.Type() {
		return newValidationError(path)
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() != w.Bool() {
			return newValidationError(path)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64:
		if v.Int() != w.Int() {
			return newValidationError(path)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		if v.Uint() != w.Uint() {
			return newValidationError(path)
		}

	case reflect.Float32, reflect.Float64:
		if v.Float() != w.Float() {
			return newValidationError(path)
		}

	case reflect.Complex64, reflect.Complex128:
		if v.Complex() != w.Complex() {
			return newValidationError(path)
		}

	case reflect.String:
		if ignoreCase {
			if !strings.EqualFold(v.String(), w.String()) {
				return newValidationError(path)
			}
		} else {
			if v.String() != w.String() {
				return newValidationError(path)
			}
		}

	case reflect.Slice:
		if v.IsNil() != w.IsNil() {
			return newValidationError(path)
		}

		fallthrough

	case reflect.Array:
		if v.Len() != w.Len() {
			return newValidationError(path)
		}

		for i := 0; i < v.Len(); i++ {
			index := fmt.Sprintf("[%d]", i)
			if v.Index(i).Kind() == reflect.Struct {
				f := v.Index(i).FieldByName("Name")
				if f.Kind() == reflect.String {
					index = fmt.Sprintf("['%s']", f.String())
				}
			}

			err := validate(path+index, v.Index(i), w.Index(i), ignoreCase)
			if err != nil {
				return err
			}
		}

	case reflect.Interface, reflect.Ptr:
		if v.IsNil() != w.IsNil() {
			return newValidationError(path)
		}

		if v.IsNil() {
			return nil
		}

		err := validate(path, v.Elem(), w.Elem(), ignoreCase)
		if err != nil {
			return err
		}

	case reflect.Map:
		if v.IsNil() != w.IsNil() {
			return newValidationError(path)
		}

		if v.Len() != w.Len() {
			return newValidationError(path)
		}

		i := v.MapRange()
		for i.Next() {
			k := i.Key()

			mapW := w.MapIndex(k)
			if !mapW.IsValid() {
				return newValidationError(path)
			}

			err := validate(fmt.Sprintf("%s[%q]", path, k.Interface()), v.MapIndex(k), mapW, ignoreCase)
			if err != nil {
				return err
			}
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			structField := v.Type().Field(i)

			if strings.EqualFold(structField.Tag.Get("mutable"), "true") {
				continue
			}

			name := strings.SplitN(structField.Tag.Get("json"), ",", 2)[0]
			if name == "" {
				name = structField.Name
			}

			subpath := path
			if subpath != "" {
				subpath += "."
			}
			subpath += name

			// Read-only properties should be omitted from PUT/POST requests.
			if strings.EqualFold(structField.Tag.Get("swagger"), "readOnly") {
				if !v.FieldByIndex([]int{i}).IsZero() {
					return newValidationError(subpath)
				}
				continue
			}

			ic := ignoreCase || strings.EqualFold(structField.Tag.Get("mutable"), "case")

			err := validate(subpath, v.Field(i), w.Field(i), ic)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unimplemented kind %s", v.Kind())
	}

	return nil
}

func newValidationError(path string) error {
	return &ValidationError{
		Target:  path,
		Message: fmt.Sprintf("Changing property '%s' is not allowed.", path),
	}
}
