package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (mon *monitor) emitAPIServerHealthCode(ctx context.Context, cli kubernetes.Interface, oc *api.OpenShiftCluster) (int, error) {
	var statusCode int
	err := cli.Discovery().RESTClient().
		Get().
		Context(ctx).
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()

	mon.clusterm.EmitGauge(metricAPIServerHealthCode, 1, map[string]string{
		"resource": oc.ID,
		"code":     strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}
