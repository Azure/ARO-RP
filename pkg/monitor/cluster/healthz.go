package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"github.com/Azure/go-autorest/autorest/azure"
)

func (mon *Monitor) emitAPIServerHealthzCode(ctx context.Context) (int, error) {
	r, err := azure.ParseResourceID(mon.oc.ID)
	if err != nil {
		return 0, err
	}

	var statusCode int
	err = mon.cli.Discovery().RESTClient().
		Get().
		Context(ctx).
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()

	mon.m.EmitGauge("apiserver.healthz.code", 1, map[string]string{
		"resourceID":     mon.oc.ID,
		"subscriptionID": r.SubscriptionID,
		"resourceGroup":  r.ResourceGroup,
		"resourceName":   r.ResourceName,
		"code":           strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}
