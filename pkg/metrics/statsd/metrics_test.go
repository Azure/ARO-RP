package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
)

func TestMarshalFloat(t *testing.T) {
	f := metric{
		Metric:    "metric",
		Namespace: "namespace",
		Dims:      map[string]string{"key": "value"},

		TS:         time.Unix(0, 0),
		ValueFloat: to.Float64Ptr(1.0),
	}
	b, err := f.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Metric":"metric","Namespace":"namespace","Dims":{"key":"value"},"TS":"1970-01-01T00:00:00.000"}:1.000000|f`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}

func TestMarshalGauge(t *testing.T) {
	g := metric{
		Metric:    "metric",
		Namespace: "namespace",
		Dims:      map[string]string{"key": "value"},

		TS:         time.Unix(0, 0),
		ValueGauge: to.Int64Ptr(42),
	}
	b, err := g.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Metric":"metric","Namespace":"namespace","Dims":{"key":"value"},"TS":"1970-01-01T00:00:00.000"}:42|g`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}
