package immutable

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/jim-minter/rp/pkg/api"
)

// Validate returns nil if v and w are identical, bar any differences on any
// struct fields explicitly tagged `mutable:"true"`.  Otherwise it returns a
// CloudError indicating the first difference it finds
func Validate(path string, v, w interface{}) error {
	return validate(path, reflect.ValueOf(v), reflect.ValueOf(w), false)
}

func validate(path string, v, w reflect.Value, ignoreCase bool) error {
	if v.Type() != w.Type() {
		return validationError(path)
	}

	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128:
		if v.Interface() != w.Interface() {
			return validationError(path)
		}

	case reflect.String:
		if ignoreCase {
			if !strings.EqualFold(v.String(), w.String()) {
				return validationError(path)
			}
		} else {
			if v.String() != w.String() {
				return validationError(path)
			}
		}

	case reflect.Slice:
		if v.IsNil() != w.IsNil() {
			return validationError(path)
		}

		fallthrough

	case reflect.Array:
		if v.Len() != w.Len() {
			return validationError(path)
		}

		for i := 0; i < v.Len(); i++ {
			err := validate(fmt.Sprintf("%s[%d]", path, i), v.Index(i), w.Index(i), ignoreCase)
			if err != nil {
				return err
			}
		}

	case reflect.Interface, reflect.Ptr:
		if v.IsNil() != w.IsNil() {
			return validationError(path)
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
			return validationError(path)
		}

		if v.Len() != w.Len() {
			return validationError(path)
		}

		i := v.MapRange()
		for i.Next() {
			k := i.Key()

			err := validate(fmt.Sprintf("%s[%q]", path, k.Interface()), v.MapIndex(k), w.MapIndex(k), ignoreCase)
			if err != nil {
				return err
			}
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if strings.EqualFold(v.Type().Field(i).Tag.Get("mutable"), "true") {
				continue
			}

			name := strings.SplitN(v.Type().Field(i).Tag.Get("json"), ",", 2)[0]
			if name == "" {
				name = v.Type().Field(i).Name
			}

			subpath := path
			if subpath != "" {
				subpath += "."
			}
			subpath += name

			ic := ignoreCase || strings.EqualFold(v.Type().Field(i).Tag.Get("mutable"), "case")

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

func validationError(path string) error {
	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path, fmt.Sprintf("Changing property '%s' is not allowed.", path))
}
