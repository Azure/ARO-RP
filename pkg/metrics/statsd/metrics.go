package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// metric represents generic metric structure
type metric struct {
	Metric    string
	Account   string
	Namespace string
	Dims      map[string]string
	TS        time.Time

	ValueGauge *int64
	ValueFloat *float64
}

// MarshalJSON marshals a metric into JSON format.
func (f *metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Metric    string
		Account   string `json:"Account,omitempty"`
		Namespace string `json:"Namespace,omitempty"`
		Dims      map[string]string
		TS        string
	}{
		Metric:    f.Metric,
		Account:   f.Account,
		Namespace: f.Namespace,
		Dims:      f.Dims,
		TS:        f.TS.UTC().Format("2006-01-02T15:04:05.000"),
	})
}

// MarshalStatsd a metric into its statsd format. Call this instead of MarshalJSON().
func (f *metric) MarshalStatsd() ([]byte, error) {
	buf := &bytes.Buffer{}

	e := json.NewEncoder(buf)
	err := e.Encode(f)
	if err != nil {
		return nil, err
	}

	// json.Encoder.Encode() appends a "\n" that we don't want - remove it
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}

	if f.ValueFloat != nil {
		_, err = fmt.Fprintf(buf, ":%f|f\n", *f.ValueFloat)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = fmt.Fprintf(buf, ":%d|g\n", *f.ValueGauge)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
