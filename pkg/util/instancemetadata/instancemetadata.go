package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

type InstanceMetadata interface {
	TenantID() string
	SubscriptionID() string
	Location() string
	ResourceGroup() string
}

type instanceMetadata struct {
	tenantID       string
	subscriptionID string
	location       string
	resourceGroup  string
}

func (im *instanceMetadata) TenantID() string {
	return im.tenantID
}

func (im *instanceMetadata) SubscriptionID() string {
	return im.subscriptionID
}

func (im *instanceMetadata) Location() string {
	return im.location
}

func (im *instanceMetadata) ResourceGroup() string {
	return im.resourceGroup
}

func New(ctx context.Context, deploymentMode deployment.Mode) (InstanceMetadata, error) {
	if deploymentMode == deployment.Development {
		return NewDev()
	}

	return newProd(ctx)
}
