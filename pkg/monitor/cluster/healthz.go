package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strconv"
)

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

func (mon *Monitor) emitAPIServerHealthzCode(statusCode int) {
	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})
}
