package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// PersistedGraph is a graph read from the cluster storage account.
// Unfortunately as the object schema changes over time, there are no guarantees
// that we can easily parse the objects in the graph, so we leave them as json
// RawMessages. You can expect Get() to work in the context of cluster creation,
// but not necessarily subsequently.

type PersistedGraph map[string]json.RawMessage

func (pg PersistedGraph) Get(disallowUnknownFields bool, is ...interface{}) error {
	for _, i := range is {
		d := json.NewDecoder(bytes.NewReader(pg[reflect.TypeOf(i).Elem().String()]))

		if disallowUnknownFields {
			d.DisallowUnknownFields()
		}

		err := d.Decode(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pg PersistedGraph) GetByName(disallowUnknownFields bool, name string, out interface{}) error {
	graphItem, ok := pg[name]
	if !ok {
		return fmt.Errorf("entry '%s' not found in persisted graph", name)
	}

	d := json.NewDecoder(bytes.NewReader(graphItem))
	if disallowUnknownFields {
		d.DisallowUnknownFields()
	}

	return d.Decode(out)
}
