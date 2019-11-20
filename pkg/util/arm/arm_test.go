package arm

import (
	"encoding/json"
	"testing"
)

func TestResource(t *testing.T) {
	tests := []struct {
		name     string
		input    *Resource
		expected string
	}{
		{
			name: "non-zero values",
			input: &Resource{
				Name: "test",
				Resource: ResourceWithCustomMarshaling{
					MapField:          map[string]int{"zero": 0, "one": 1},
					SliceField:        []int{0, 1},
					SliceOfBytesField: []byte("test"),
					ArrayField:        [2]int{0, 1},
					InterfaceField:    1,
					PtrField:          stringPtr("test"),
					unexportedField:   1,
					BoolField:         true,
					IntField:          1,
					UintField:         1,
					FloatField:        1,
					StringField:       "test",
					StructField:       NestedStruct{NestedField: 1},
				},
			},
			expected: `{
 "array_field": [
  0,
  1
 ],
 "bool_field": true,
 "float_field": 1,
 "int_field": 1,
 "interface_field": 1,
 "map_field": {
  "one": 1,
  "zero": 0
 },
 "name": "test",
 "ptr_field": "test",
 "slice_field": [
  0,
  1
 ],
 "slice_of_bytes_field": "dGVzdA==",
 "string_field": "test",
 "struct_field": {
  "nested_field": 1
 },
 "uint_field": 1
}`,
		},
		{
			name: "zero values",
			input: &Resource{
				Name:     "test",
				Resource: ResourceWithCustomMarshaling{},
			},
			expected: `{
 "array_field": [
  0,
  0
 ],
 "name": "test"
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := json.MarshalIndent(test.input, "", " ")
			if err != nil {
				t.Errorf("unexpected error: %s", err.Error())
			}

			result := string(b)
			if result != test.expected {
				t.Errorf("got %s, expected %s", result, test.expected)
			}
		})
	}
}

type ResourceWithCustomMarshaling struct {
	MapField          map[string]int `json:"map_field,omitempty"`
	SliceField        []int          `json:"slice_field,omitempty"`
	SliceOfBytesField []byte         `json:"slice_of_bytes_field,omitempty"`
	ArrayField        [2]int         `json:"array_field,omitempty"`
	InterfaceField    interface{}    `json:"interface_field,omitempty"`
	PtrField          *string        `json:"ptr_field,omitempty"`
	BoolField         bool           `json:"bool_field,omitempty"`
	IntField          int            `json:"int_field,omitempty"`
	UintField         uint           `json:"uint_field,omitempty"`
	FloatField        float64        `json:"float_field,omitempty"`
	StringField       string         `json:"string_field,omitempty"`
	StructField       NestedStruct   `json:"struct_field,omitempty"`
	unexportedField   int
}

// MarshalJSON contains custom marshaling logic which we expect to be dropped
// during marshalling as part of arm.Resource type
func (r *ResourceWithCustomMarshaling) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if r.MapField != nil {
		objectMap["custom_field_key"] = r.MapField
	}
	return json.Marshal(objectMap)
}

type NestedStruct struct {
	NestedField int `json:"nested_field,omitempty"`
}

// stringPtr returns a pointer to the passed string.
func stringPtr(s string) *string {
	return &s
}
