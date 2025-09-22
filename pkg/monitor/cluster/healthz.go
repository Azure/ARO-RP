package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
)

func (mon *Monitor) emitAPIServerHealthzCode(ctx context.Context) error {
	var statusCode int
	err := mon.rawClient.
		Get().
		AbsPath("/healthz").
		Do(ctx).
		StatusCode(&statusCode).
		Error()

	mon.emitGauge("apiserver.healthz.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	if err != nil {
		return errors.Join(errAPIServerHealthzFailure, err)
	}

	return err
}

func (mon *Monitor) emitAPIServerPingCode(ctx context.Context) error {
	var statusCode int
	err := mon.rawClient.
		Get().
		AbsPath("/healthz/ping").
		Do(ctx).
		StatusCode(&statusCode).
		Error()

	mon.emitGauge("apiserver.healthz.ping.code", 1, map[string]string{
		"code": strconv.FormatInt(int64(statusCode), 10),
	})

	if err != nil {
		return errors.Join(errAPIServerPingFailure, err)
	}

	return err
}
