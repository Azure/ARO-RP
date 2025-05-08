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
	name       string
	account    string
	namespace  string
	dimensions map[string]string
	timestamp  time.Time

	valueGauge *int64
	valueFloat *float64
}

// MarshalJSON marshals a metric into JSON format.
func (m *metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Metric    string            `json:"Metric"`
		Account   string            `json:"Account,omitempty"`
		Namespace string            `json:"Namespace,omitempty"`
		Dims      map[string]string `json:"Dims"`
		TS        string            `json:"TS"`
	}{
		Metric:    m.name,
		Account:   m.account,
		Namespace: m.namespace,
		Dims:      m.dimensions,
		TS:        m.timestamp.UTC().Format("2006-01-02T15:04:05.000"),
	})
}

// marshalStatsd marshals a metric into its statsd format. Call this instead of
// MarshalJSON().
func (m *metric) marshalStatsd() ([]byte, error) {
	buf := &bytes.Buffer{}

	e := json.NewEncoder(buf)
	err := e.Encode(m)
	if err != nil {
		return nil, err
	}

	// json.Encoder.Encode() appends a "\n" that we don't want - remove it
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}

	if m.valueFloat != nil {
		_, err = fmt.Fprintf(buf, ":%f|f\n", *m.valueFloat)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = fmt.Fprintf(buf, ":%d|g\n", *m.valueGauge)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
