package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strconv"
)

func (mon *Monitor) initialHealthChecks(ctx context.Context) (skipRemaining bool) {
	statusCode, err := mon.getAPIServerHealthzCode()
	if err != nil {
		mon.logAndEmitError(mon.getAPIServerHealthzCode, err)
	}
	if statusCode != http.StatusOK {
		skipHealthz := mon.emitDiagnosis(ctx)
		if !skipHealthz {
			mon.emitAPIServerHealthzCode(statusCode)
		}
		skipRemaining = true
	} else {
		mon.emitAPIServerHealthzCode(statusCode)
	}
	return
}

func (mon *Monitor) getAPIServerHealthzCode() (int, error) {
	var statusCode int
	err := mon.cli.Discovery().RESTClient().
		Get().
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()
	return statusCode, err
}

type apiServerDiagnostic func(context.Context) (bool, error)

// emitDiagnosis looks for a cause of a non-200 API server status code.
// If it finds one, it will be emitted as a separate metric.
// It also specifies whether the API server healthz metric should be skipped,
// so as to avoid raising an alert when it's not necessary (e.g. VMs have been
// manually powered off in Azure).
func (mon *Monitor) emitDiagnosis(ctx context.Context) (skipHealthz bool) {
	for _, f := range []apiServerDiagnostic{
		mon.emitStoppedVMPowerStatus,
		// place additional checks here
	} {
		foundCause, err := f(ctx)
		if err != nil {
			mon.logAndEmitError(f, err)
		}
		if foundCause {
			skipHealthz = true
		}
		// continue diagnosing without unsetting skipHealthz flag (once there are more checks)
	}
	return
}

func (mon *Monitor) emitAPIServerHealthzCode(statusCode int) {
	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})
}
