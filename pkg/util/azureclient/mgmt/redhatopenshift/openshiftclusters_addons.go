package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift"
)

// OpenShiftClustersClientAddons contains addons for OpenShiftClustersClient
type OpenShiftClustersClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters redhatopenshift.OpenShiftCluster) error
}

func (c *openShiftClustersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters redhatopenshift.OpenShiftCluster) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, resourceName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
