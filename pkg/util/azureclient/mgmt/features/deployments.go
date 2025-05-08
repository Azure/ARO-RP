package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DeploymentsClient is a minimal interface for azure DeploymentsClient
type DeploymentsClient interface {
	Get(ctx context.Context, resourceGroupName, deploymentName string) (mgmtfeatures.DeploymentExtended, error)
	DeploymentsClientAddons
}

type deploymentsClient struct {
	mgmtfeatures.DeploymentsClient
}

var _ DeploymentsClient = &deploymentsClient{}

// NewDeploymentsClient creates a new DeploymentsClient
func NewDeploymentsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DeploymentsClient {
	client := mgmtfeatures.NewDeploymentsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.PollingDelay = 10 * time.Second
	client.PollingDuration = time.Hour
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &deploymentsClient{
		DeploymentsClient: client,
	}
}
