package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openshift/installer/pkg/asset"
)

func (g graph) MarshalJSON() ([]byte, error) {
	m := map[string]asset.Asset{}
	for t, a := range g {
		m[t.String()] = a
	}
	return json.Marshal(m)
}

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
		t, found := registeredTypes[n]
		if !found {
			return fmt.Errorf("unregistered type %q", n)
		}

		a := reflect.New(reflect.TypeOf(t).Elem()).Interface().(asset.Asset)
		err = json.Unmarshal(b, a)
		if err != nil {
			return err
		}

		(*g)[reflect.TypeOf(a)] = a
	}

	return nil
}
