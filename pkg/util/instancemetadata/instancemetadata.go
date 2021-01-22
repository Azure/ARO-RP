package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

type InstanceMetadata interface {
	Hostname() string
	TenantID() string
	SubscriptionID() string
	Location() string
	ResourceGroup() string
	Environment() *azure.Environment
}

type instanceMetadata struct {
	hostname       string
	tenantID       string
	subscriptionID string
	location       string
	resourceGroup  string
	environment    *azure.Environment
}

func (im *instanceMetadata) Hostname() string {
	return im.hostname
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

func (im *instanceMetadata) Environment() *azure.Environment {
	return im.environment
}

func New(ctx context.Context, deploymentMode deployment.Mode) (InstanceMetadata, error) {
	if deploymentMode == deployment.Development {
		return NewDev(true)
	}

	return newProd(ctx)
}

func NewClusterMetadata(env InstanceMetadata) InstanceMetadata {
	// awful heuristics, remove when the value of CLUSTER_RESOURCEGROUP in the
	// pipelines has been updated to the cluster's RG and we can assume
	// RESOURCEGROUP is the CI environment.
	clusterRG, exists := os.LookupEnv("CLUSTER_RESOURCEGROUP")
	if !exists {
		clusterRG = os.Getenv("CLUSTER")
	}

	return &instanceMetadata{
		hostname:       env.Hostname(),
		tenantID:       env.TenantID(),
		subscriptionID: env.SubscriptionID(),
		location:       env.Location(),
		resourceGroup:  clusterRG,
		environment:    env.Environment(),
	}
}
