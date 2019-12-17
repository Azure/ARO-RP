package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
)

func TestResourceMarshal(t *testing.T) {
	tests := []struct {
		name string
		r    *Resource
		want []byte
	}{
		{
			name: "non-zero values",
			r: &Resource{
				Name: "test",
				Resource: &testResource{
					Bool:      true,
					Int:       1,
					Uint:      1,
					Float:     1.1,
					Array:     [1]*testResource{{Bool: true, Unmarshaled: 1}},
					Interface: &testResource{Int: 1, Unmarshaled: 1},
					Map: map[string]*testResource{
						"zero": {Uint: 0, Unmarshaled: 1},
						"one":  {Uint: 1, Unmarshaled: 1},
					},
					Ptr:         to.StringPtr("test"),
					Slice:       []*testResource{{Float: 1.1, Unmarshaled: 1}},
					ByteSlice:   []byte("test"),
					String:      "test",
					Struct:      &testResource{String: "test", Unmarshaled: 1},
					Name:        "should be overwritten by parent name",
					Unmarshaled: 1,
					unexported:  1,
				},
			},
			want: []byte(`{
    "bool": true,
    "int": 1,
    "uint": 1,
    "float": 1.1,
    "array": [
        {
            "bool": true,
            "tags": null
        }
    ],
    "interface": {
        "int": 1,
        "tags": null
    },
    "map": {
        "one": {
            "uint": 1,
            "tags": null
        },
        "zero": {
            "tags": null
        }
    },
    "ptr": "test",
    "slice": [
        {
            "float": 1.1,
            "tags": null
        }
    ],
    "byte_slice": "dGVzdA==",
    "string": "test",
    "struct": {
        "string": "test",
        "tags": null
    },
    "name": "test"
}`),
		},
		{
			name: "zero values",
			r: &Resource{
				Name:     "test",
				Resource: &testResource{},
			},
			want: []byte(`{
    "name": "test"
}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := json.MarshalIndent(test.r, "", "    ")
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(b, test.want) {
				t.Error(string(b))
			}
		})
	}
}

type testResource struct {
	Bool        bool                     `json:"bool,omitempty"`
	Int         int                      `json:"int,omitempty"`
	Uint        uint                     `json:"uint,omitempty"`
	Float       float64                  `json:"float,omitempty"`
	Array       [1]*testResource         `json:"array,omitempty"`
	Interface   interface{}              `json:"interface,omitempty"`
	Map         map[string]*testResource `json:"map,omitempty"`
	Ptr         *string                  `json:"ptr,omitempty"`
	Slice       []*testResource          `json:"slice,omitempty"`
	ByteSlice   []byte                   `json:"byte_slice,omitempty"`
	String      string                   `json:"string,omitempty"`
	Struct      *testResource            `json:"struct,omitempty"`
	Name        string                   `json:"name,omitempty"`
	Unmarshaled int                      `json:"-"`
	unexported  int
	// Both `arm.Resource` and nested `testResource` have fields with name `Tags`.
	// The `Tags` field from `arm.Resource` must override the one from `testResource`
	// on the top-level of JSON.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON contains custom marshaling logic which we expect to be dropped
// during marshalling as part of arm.Resource type
func (r *testResource) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("should not be called")
}
