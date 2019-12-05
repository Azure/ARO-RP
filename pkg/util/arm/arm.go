package arm

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	uuid "github.com/satori/go.uuid"
)

// Template represents an ARM template
type Template struct {
	Schema         string                 `json:"$schema,omitempty"`
	APIProfile     string                 `json:"apiProfile,omitempty"`
	ContentVersion string                 `json:"contentVersion,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
	Parameters     map[string]*Parameter  `json:"parameters,omitempty"`
	Functions      []interface{}          `json:"functions,omitempty"`
	Resources      []*Resource            `json:"resources,omitempty"`
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

// MarshalJSON marshals the nested r.Resource ignoring any MarshalJSON() methods
// on its types.  It then merges remaining fields of r over the result
func (r *Resource) MarshalJSON() ([]byte, error) {
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

		if idx, found := indexes[field.Name]; found {
			field.Type = emptyInterfaceType
			fields[idx] = field
			if !isZero(outer.Field(i)) {
				values[idx] = outer.Field(i)
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
		if v.Type() == reflect.TypeOf(uuid.UUID{}) {
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
			if !isZero(v) {
				s.Field(i).Set(v)
			}
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
