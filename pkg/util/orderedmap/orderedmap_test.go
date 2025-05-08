package orderedmap

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshalDuplicateField(t *testing.T) {
	in := []byte(`{"a":1,"b":0,"b":2}`)
	out := keyValues{
		{
			Key:   "a",
			Value: 1,
		},
		{
			Key:   "b",
			Value: 2,
		},
	}

	var m keyValues
	err := json.Unmarshal(in, &m)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, out) {
		t.Error(m)
	}
}
