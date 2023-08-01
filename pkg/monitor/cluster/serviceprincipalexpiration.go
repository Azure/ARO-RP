package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
)

/****************************************************
	Monitor the Service Prinicpal exipiration date
****************************************************/

func (mon *Monitor) emitSpExpiration(ctx context.Context) error {
	expiryDate, err := utilgraph.GetServicePrincipalExpiryByAppID(ctx, mon.spGraphClient, mon.oc.Properties.ServicePrincipalProfile.ClientID)
	if err != nil {
		return err
	}

	mon.emitGauge("cluster.serviceprincipal.expiration", 1, map[string]string{
		"expiryDate": expiryDate.String(),
	})

	return nil
}
