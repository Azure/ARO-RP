package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (mon *monitor) validateAPIHealth(ctx context.Context, cli kubernetes.Interface, oc *api.OpenShiftCluster) error {
	var statusCode int
	err := cli.Discovery().RESTClient().
		Get().
		Context(ctx).
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()
	if err != nil && statusCode == 0 {
		return fmt.Errorf("API Server is Unhealthy - %s", err.Error())
	}

	mon.clusterm.EmitGauge(MetricAPIServerHeatlth, 1, map[string]string{
		"resource": oc.ID,
		"code":     strconv.FormatInt(int64(statusCode), 10),
	})

	return nil
}
