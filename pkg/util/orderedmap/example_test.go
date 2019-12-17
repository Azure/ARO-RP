package orderedmap

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

type keyValue struct {
	Key   string
	Value int
}

type keyValues []keyValue

func (xs *keyValues) UnmarshalJSON(b []byte) error {
	return UnmarshalJSON(b, xs)
}

func (xs keyValues) MarshalJSON() ([]byte, error) {
	return MarshalJSON(xs)
}

func TestExample(t *testing.T) {
	in := []byte(`{"a":1,"b":2}`)
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

	b, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, in) {
		t.Error(string(b))
	}
}
