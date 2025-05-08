package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
)

var _ http.RoundTripper = (*tracerRoundTripper)(nil)

type tracerRoundTripper struct {
	log *logrus.Entry
	m   metrics.Emitter
	tr  http.RoundTripper
}

func New(log *logrus.Entry, tr *http.Transport, m metrics.Emitter) *tracerRoundTripper {
	return &tracerRoundTripper{
		log: log,
		m:   m,
		tr:  tr,
	}
}

func (t *tracerRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	start := time.Now()

	defer func() {
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) >= 2 && parts[len(parts)-2] == "docs" {
			parts[len(parts)-1] = "{id}"
		}
		path := strings.Join(parts, "/")

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		t.m.EmitGauge("client.cosmosdb.count", 1, map[string]string{
			"code": strconv.Itoa(statusCode),
			"verb": req.Method,
			"path": path,
		})

		t.m.EmitGauge("client.cosmosdb.duration", time.Since(start).Milliseconds(), map[string]string{
			"code": strconv.Itoa(statusCode),
			"verb": req.Method,
			"path": path,
		})

		if err != nil {
			t.m.EmitGauge("client.cosmosdb.errors", 1, map[string]string{
				"code": strconv.Itoa(statusCode),
				"verb": req.Method,
				"path": path,
			})
		}

		if resp != nil {
			// Sometimes we get request-charge="" because pkranges API is free
			requestCharge := strings.Trim(resp.Header.Get("x-ms-request-charge"), `"`)

			ru, parseErr := strconv.ParseFloat(requestCharge, 64)
			if parseErr == nil {
				t.m.EmitFloat("client.cosmosdb.requestunits", ru, map[string]string{
					"code": strconv.Itoa(statusCode),
					"verb": req.Method,
					"path": path,
				})
			}
		}
	}()

	return t.tr.RoundTrip(req)
}
