package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
)

func (mon *Monitor) emitAPIServerHealthzCode(ctx context.Context) (int, error) {
	var statusCode int
	err := mon.cli.Discovery().RESTClient().
		Get().
		AbsPath("/healthz").
		Do(ctx).
		StatusCode(&statusCode).
		Error()

	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}

func (mon *Monitor) emitAPIServerPingCode(ctx context.Context) error {
	var statusCode int
	err := mon.cli.Discovery().RESTClient().
		Get().
		AbsPath("/healthz/ping").
		Do(ctx).
		StatusCode(&statusCode).
		Error()

	mon.emitGauge("apiserver.healthz.ping.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	return err
}
