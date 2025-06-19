package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestMarshalFloat(t *testing.T) {
	f := metric{
		name:       "metric",
		namespace:  "namespace",
		dimensions: map[string]string{"key": "value"},

		timestamp:  time.Unix(0, 0),
		valueFloat: pointerutils.ToPtr(1.0),
	}
	b, err := f.marshalStatsd()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Metric":"metric","Namespace":"namespace","Dims":{"key":"value"},"TS":"1970-01-01T00:00:00.000"}:1.000000|f`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}

func TestMarshalGauge(t *testing.T) {
	g := metric{
		name:       "metric",
		namespace:  "namespace",
		dimensions: map[string]string{"key": "value"},

		timestamp:  time.Unix(0, 0),
		valueGauge: pointerutils.ToPtr(int64(42)),
	}
	b, err := g.marshalStatsd()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Metric":"metric","Namespace":"namespace","Dims":{"key":"value"},"TS":"1970-01-01T00:00:00.000"}:42|g`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}
