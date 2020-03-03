package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (mon *monitor) emitAPIServerHealthzCode(ctx context.Context, cli kubernetes.Interface, oc *api.OpenShiftCluster) (int, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return 0, err
	}

	var statusCode int
	err = cli.Discovery().RESTClient().
		Get().
		Context(ctx).
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()

	mon.clusterm.EmitGauge("apiserver.healthz.code", 1, map[string]string{
		"resourceID":     oc.ID,
		"subscriptionID": r.SubscriptionID,
		"resourceGroup":  r.ResourceGroup,
		"resourceName":   r.ResourceName,
		"code":           strconv.FormatInt(int64(statusCode), 10),
	})

	return statusCode, err
}
