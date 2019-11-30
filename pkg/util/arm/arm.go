package arm

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/jim-minter/rp/pkg/util/orderedmap"
)

// Template represents an ARM template
type Template struct {
	Schema         string                 `json:"$schema,omitempty"`
	APIProfile     string                 `json:"apiProfile,omitempty"`
	ContentVersion string                 `json:"contentVersion,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
	Parameters     map[string]Parameter   `json:"parameters,omitempty"`
	Functions      []interface{}          `json:"functions,omitempty"`
	Resources      []Resource             `json:"resources,omitempty"`
	Outputs        map[string]interface{} `json:"outputs,omitempty"`
}

// Parameter represents an ARM template parameter
type Parameter struct {
	Type          string                 `json:"type,omitempty"`
	DefaultValue  interface{}            `json:"defaultValue,omitempty"`
	AllowedValues []interface{}          `json:"allowedValues,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	MinValue      int                    `json:"minValue,omitempty"`
	MaxValue      int                    `json:"maxValue,omitempty"`
	MinLength     int                    `json:"minLength,omitempty"`
	MaxLength     int                    `json:"maxLength,omitempty"`
}

// Resource represents an ARM template resource
type Resource struct {
	Resource interface{}

	Name       string                 `json:"name,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Condition  bool                   `json:"condition,omitempty"`
	APIVersion string                 `json:"apiVersion,omitempty"`
	DependsOn  []string               `json:"dependsOn,omitempty"`
	Location   string                 `json:"location,omitempty"`
	Tags       map[string]interface{} `json:"tags,omitempty"`
	Copy       *Copy                  `json:"copy,omitempty"`
	Comments   string                 `json:"comments,omitempty"`
}

// Copy represents an ARM template copy stanza
type Copy struct {
	Name      string `json:"name,omitempty"`
	Count     int    `json:"count,omitempty"`
	Mode      string `json:"mode,omitempty"`
	BatchSize int    `json:"batchSize,omitempty"`
}

type keyValue struct {
	Key   string
	Value interface{}
}

type keyValues []keyValue

func (xs *keyValues) UnmarshalJSON(b []byte) error {
	return orderedmap.UnmarshalJSON(b, xs)
}

func (xs keyValues) MarshalJSON() ([]byte, error) {
	return orderedmap.MarshalJSON(xs)
}

// MarshalJSON marshals the nested r.Resource ignoring any MarshalJSON() methods
// on its types.  It then merges remaining fields of r over the result
func (r Resource) MarshalJSON() ([]byte, error) {
	// first marshal the nested r.Resource ignoring any MarshalJSON() methods
	b, err := json.Marshal(shadowCopy(r.Resource))
	if err != nil {
		return nil, err
	}

	var nested keyValues

	err = json.Unmarshal(b, &nested)
	if err != nil {
		return nil, err
	}

	// now create a shadow of r without the nested r.Resource field, and marshal
	// that
	v := reflect.ValueOf(r)

	fields := make([]reflect.StructField, 0, v.NumField()-1)
	for i := 0; i < v.NumField()-1; i++ {
		fields = append(fields, v.Type().Field(i+1))
	}

	shadow := reflect.New(reflect.StructOf(fields)).Elem()

	for i := 0; i < v.NumField()-1; i++ {
		shadow.Field(i).Set(v.Field(i + 1))
	}

	b, err = json.Marshal(shadow.Interface())
	if err != nil {
		return nil, err
	}

	var outer keyValues

	err = json.Unmarshal(b, &outer)
	if err != nil {
		return nil, err
	}

	// now scan through nested and remove any keys that appear in outer
	outerkeys := map[string]struct{}{}
	for _, kv := range outer {
		outerkeys[kv.Key] = struct{}{}
	}

	{
		newnested := make(keyValues, 0, len(nested))
		for _, kv := range nested {
			if _, found := outerkeys[kv.Key]; !found {
				newnested = append(newnested, kv)
			}
		}
		nested = newnested
	}

	// finally, append outer to nested, marshal and return
	nested = append(nested, outer...)

	return json.Marshal(nested)
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
		a := reflect.New(reflect.ArrayOf(v.Len(), emptyInterfaceType)).Elem()
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
			t = reflect.SliceOf(v.Type().Elem())
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
		for i := 0; i < v.Type().NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			field := v.Type().Field(i)
			field.Type = emptyInterfaceType
			fields = append(fields, field)
		}
		t := reflect.StructOf(fields)

		s := reflect.New(t).Elem()
		for i, j := 0, 0; i < v.NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			f := _shadowCopy(v.Field(i))
			if !isZero(f) {
				s.Field(j).Set(f)
			}
			j++
		}
		return s

	default:
		return v
	}
}

// isZero is a copy of `func (v reflect.Value) IsZero() bool`, which is built-in
// in from Go 1.13
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// This should never happen, but will act as a safeguard for
		// later, as a default value doesn't makes sense here.
		panic(&reflect.ValueError{Method: "reflect.Value.IsZero", Kind: v.Kind()})
	}
}
