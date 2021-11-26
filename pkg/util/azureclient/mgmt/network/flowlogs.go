package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// FlowLogsClient is a minimal interface for azure FlowLogsClient
type FlowLogsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string) (result mgmtnetwork.FlowLog, err error)

	FlowLogsClientAddons
}

type flowLogsClient struct {
	mgmtnetwork.FlowLogsClient
}

var _ FlowLogsClient = &flowLogsClient{}

// NewFlowLogsClient creates a new FlowLogsClient
func NewFlowLogsClient(environment *azureclient.AROEnvironment, tenantID string, authorizer autorest.Authorizer) FlowLogsClient {
	client := mgmtnetwork.NewFlowLogsClientWithBaseURI(environment.ResourceManagerEndpoint, tenantID)
	client.Authorizer = authorizer
	return &flowLogsClient{
		FlowLogsClient: client,
	}
}
