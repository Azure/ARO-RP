package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strconv"
	"strings"
)

func (mon *Monitor) emitAPIServerHealthzCode() (int, error) {
	var statusCode int
	raw, err := mon.cli.Discovery().RESTClient().
		Get().
		AbsPath("/readyz").
		Do().
		StatusCode(&statusCode).
		Raw()

	if statusCode == 200 && err == nil {
		// apiserver is healthy
		mon.emitGauge("apiserver.ready", 1, nil)
	} else if err == nil {
		// apiserver is unhealthy, but is reporting what's wrong
		var content strings.Builder
		content.Write(raw)
		failures := strings.Split(content.String(), "\n")

		for _, failure := range failures {
			if strings.HasPrefix(failure, "[-]") {
				failedUnit := failure[3:strings.Index(failure, " ")]
				mon.emitGauge("apiserver.ready", 0, map[string]string{
					"failedUnit": failedUnit,
				})
			}
		}

	} else {
		// apiserver cannot be reached
		mon.emitGauge("apiserver.ready", 0, nil)
	}

	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}
