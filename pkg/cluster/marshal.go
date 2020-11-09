package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
)

func (g *graph) UnmarshalJSON(b []byte) error {
	if *g == nil {
		*g = graph{}
	}

	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	for n, b := range m {
		var i interface{}

		if t := registeredTypes[n]; t != nil {
			i = reflect.New(reflect.TypeOf(t).Elem()).Interface()
		}

		err = json.Unmarshal(b, &i)
		if err != nil {
			return err
		}

		(*g)[n] = i
	}

	return nil
}
