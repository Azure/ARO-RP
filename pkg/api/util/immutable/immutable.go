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

// Policy describes exceptions to immutable-by-default validation for newer
// TypeSpec-generated API models.
type Policy struct {
	Mutable         []string
	ReadOnly        []string
	ReadOnlyValue   []string
	CaseInsensitive []string
	NormalizeNil    []string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validate returns nil if v and w are identical, bar any differences on any
// struct fields explicitly tagged `mutable:"true"`.  Otherwise it returns a
// CloudError indicating the first difference it finds.
//
// Used for validation in API versions before v20250725, which use hand-written
// models.
func Validate(path string, v, w interface{}) error {
	return validate(path, reflect.ValueOf(v), reflect.ValueOf(w), false)
}

// ValidateWithPolicy validates generated models without requiring struct tags.
// Scalar pointers are compared by value. Struct pointers listed in NormalizeNil
// are treated as their zero value when nil; other struct pointers retain nil as
// a meaningful value. NormalizeNil helps us handle the fact that certain struct
// fields that were previously represented using value types are now represented
// using pointer types.
//
// Used for validation in API versions on or after v20250725, which use models
// generated from TypeSpec.
func ValidateWithPolicy(path string, v, w interface{}, policy Policy) error {
	return validateWithPolicy(path, reflect.ValueOf(v), reflect.ValueOf(w), policy, false, false)
}

func validateWithPolicy(path string, v, w reflect.Value, policy Policy, ignoreCase, normalizePointer bool) error {
	if matchesAny(policy.Mutable, path) {
		return nil
	}

	if v.Type() != w.Type() {
		return newValidationError(path)
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() != w.Bool() {
			return newValidationError(path)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() != w.Int() {
			return newValidationError(path)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
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
		if ignoreCase || matchesAny(policy.CaseInsensitive, path) {
			if !strings.EqualFold(v.String(), w.String()) {
				return newValidationError(path)
			}
		} else if v.String() != w.String() {
			return newValidationError(path)
		}

	case reflect.Slice:
		if v.IsNil() != w.IsNil() || v.Len() != w.Len() {
			return newValidationError(path)
		}

		for i := 0; i < v.Len(); i++ {
			index := fmt.Sprintf("[%d]", i)
			item := v.Index(i)
			if item.Kind() == reflect.Pointer && !item.IsNil() {
				item = item.Elem()
			}
			if item.Kind() == reflect.Struct {
				field := item.FieldByName("Name")
				if field.IsValid() {
					if field.Kind() == reflect.Pointer && !field.IsNil() {
						field = field.Elem()
					}
					if field.Kind() == reflect.String {
						index = fmt.Sprintf("['%s']", field.String())
					}
				}
			}

			if err := validateWithPolicy(path+index, v.Index(i), w.Index(i), policy, ignoreCase, true); err != nil {
				return err
			}
		}

	case reflect.Array:
		if v.Len() != w.Len() {
			return newValidationError(path)
		}
		for i := 0; i < v.Len(); i++ {
			if err := validateWithPolicy(fmt.Sprintf("%s[%d]", path, i), v.Index(i), w.Index(i), policy, ignoreCase, true); err != nil {
				return err
			}
		}

	case reflect.Interface:
		if v.IsNil() != w.IsNil() {
			return newValidationError(path)
		}
		if !v.IsNil() {
			return validateWithPolicy(path, v.Elem(), w.Elem(), policy, ignoreCase, normalizePointer)
		}

	case reflect.Pointer:
		normalizePointer = normalizePointer || v.Type().Elem().Kind() != reflect.Struct || matchesAny(policy.NormalizeNil, path)
		if !normalizePointer && v.IsNil() != w.IsNil() {
			return newValidationError(path)
		}
		if v.IsNil() && w.IsNil() {
			return nil
		}

		vElem := reflect.Zero(v.Type().Elem())
		if !v.IsNil() {
			vElem = v.Elem()
		}
		wElem := reflect.Zero(w.Type().Elem())
		if !w.IsNil() {
			wElem = w.Elem()
		}
		return validateWithPolicy(path, vElem, wElem, policy, ignoreCase, false)

	case reflect.Map:
		if v.IsNil() != w.IsNil() || v.Len() != w.Len() {
			return newValidationError(path)
		}

		iterator := v.MapRange()
		for iterator.Next() {
			key := iterator.Key()
			mapW := w.MapIndex(key)
			if !mapW.IsValid() {
				return newValidationError(path)
			}
			if err := validateWithPolicy(fmt.Sprintf("%s[%q]", path, key.Interface()), iterator.Value(), mapW, policy, ignoreCase, false); err != nil {
				return err
			}
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			structField := v.Type().Field(i)
			if !structField.IsExported() {
				continue
			}

			if structField.Anonymous {
				if err := validateWithPolicy(path, v.Field(i), w.Field(i), policy, ignoreCase, false); err != nil {
					return err
				}
				continue
			}

			name := policyJSONName(structField)
			subpath := name
			if path != "" {
				subpath = path + "." + name
			}

			if matchesAny(policy.Mutable, subpath) {
				continue
			}
			if matchesAny(policy.ReadOnly, subpath) {
				fieldValue := v.Field(i)
				if matchesAny(policy.ReadOnlyValue, subpath) && fieldValue.Kind() == reflect.Pointer && !fieldValue.IsNil() {
					fieldValue = fieldValue.Elem()
				}
				if !fieldValue.IsZero() {
					return newValidationError(subpath)
				}
				continue
			}

			if err := validateWithPolicy(subpath, v.Field(i), w.Field(i), policy, ignoreCase || matchesAny(policy.CaseInsensitive, subpath), false); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unimplemented kind %s", v.Kind())
	}

	return nil
}

// matchesAny reports whether value exactly matches a pattern or matches a
// pattern containing asterisks as unrestricted substring wildcards.
func matchesAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		parts := strings.Split(pattern, "*")
		if len(parts) == 1 && pattern == value {
			return true
		}
		if len(parts) > 1 && strings.HasPrefix(value, parts[0]) && strings.HasSuffix(value, parts[len(parts)-1]) {
			position := len(parts[0])
			matched := true
			for _, part := range parts[1 : len(parts)-1] {
				index := strings.Index(value[position:], part)
				if index < 0 {
					matched = false
					break
				}
				position += index + len(part)
			}
			if matched {
				return true
			}
		}
	}
	return false
}

// policyJSONName returns the JSON property name for a struct field. It uses an
// explicit JSON tag when present and otherwise applies the casing conventions
// used by the TypeSpec-generated models.
func policyJSONName(field reflect.StructField) string {
	if tag := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]; tag != "" && tag != "-" {
		return tag
	}

	name := field.Name
	replacements := []struct{ old, new string }{
		{"IDs", "Ids"},
		{"ID", "Id"},
		{"IPs", "Ips"},
		{"IP", "Ip"},
		{"URLs", "Urls"},
		{"URL", "Url"},
	}
	for _, replacement := range replacements {
		if strings.HasSuffix(name, replacement.old) {
			name = strings.TrimSuffix(name, replacement.old) + replacement.new
			break
		}
	}
	if strings.HasPrefix(name, "VM") {
		name = "Vm" + strings.TrimPrefix(name, "VM")
	}
	return strings.ToLower(name[:1]) + name[1:]
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

	case reflect.Interface, reflect.Pointer:
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
