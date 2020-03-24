package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// DeploymentsClient is a minimal interface for azure DeploymentsClient
type DeploymentsClient interface {
	Get(ctx context.Context, resourceGroupName, deploymentName string) (mgmtresources.DeploymentExtended, error)
	GetAtSubscriptionScope(ctx context.Context, deploymentName string) (mgmtresources.DeploymentExtended, error)
	DeploymentsClientAddons
}

type deploymentsClient struct {
	mgmtresources.DeploymentsClient
}

var _ DeploymentsClient = &deploymentsClient{}

// NewDeploymentsClient creates a new DeploymentsClient
func NewDeploymentsClient(subscriptionID string, authorizer autorest.Authorizer) DeploymentsClient {
	client := mgmtresources.NewDeploymentsClient(subscriptionID)
	client.Authorizer = authorizer
	client.PollingDuration = time.Hour
	client.PollingDelay = 10 * time.Second

	return &deploymentsClient{
		DeploymentsClient: client,
	}
}
