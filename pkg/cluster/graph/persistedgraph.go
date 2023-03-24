package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
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

// Set is currently only used in unit test context.  If you want to use this in
// production, you will want to be very sure that you are not losing state that
// you may need later
func (pg PersistedGraph) Set(is ...interface{}) (err error) {
	for _, i := range is {
		pg[reflect.TypeOf(i).String()], err = json.Marshal(i)
		if err != nil {
			return err
		}
	}

	return nil
}
