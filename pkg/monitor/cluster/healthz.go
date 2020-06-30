package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
)

func (mon *Monitor) emitAPIServerHealthzCode() (int, error) {
	var statusCode int
	raw, err := mon.cli.Discovery().RESTClient().
		Get().
		AbsPath("/readyz").
		Do().
		StatusCode(&statusCode).
		Raw()

	// HTTP 500s are expected behaviour (e.g. on startup/shutdown of the API server) and should not cause this function to return an error. Therefore, silence it.
	if err != nil && errors.IsUnexpectedServerError(err) {
		err = nil
	}

	if statusCode == 200 && err == nil {
		// apiserver is ready
		mon.emitGauge("apiserver.ready", 1, nil)
	} else if statusCode != 200 && err == nil {
		// apiserver is not ready, but is still reporting status
		var emitted bool
		failures := strings.Split(string(raw), "\n")

		for _, failure := range failures {
			if strings.HasPrefix(failure, "[-]") {
				failedUnit := failure[3:strings.Index(failure, " ")]
				mon.emitGauge("apiserver.ready", 0, map[string]string{
					"failedUnit": failedUnit,
				})
				emitted = true
			}
		}

		if !emitted {
			// If we can't emit any failed units in particular, just emit a blank one
			mon.emitGauge("apiserver.ready", 0, nil)
		}
	} else {
		// apiserver cannot be reached
		mon.emitGauge("apiserver.ready", 0, nil)
	}

	// compat?
	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}
