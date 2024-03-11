package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DenyAssignmentClient is a minimal interface for azure DenyAssignmentClient
type DenyAssignmentClient interface {
	DenyAssignmentClientAddons
}

type denyAssignmentClient struct {
	mgmtauthorization.DenyAssignmentsClient
}

var _ DenyAssignmentClient = &denyAssignmentClient{}

// NewDenyAssignmentsClient creates a new DenyAssignmentsClient
func NewDenyAssignmentsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DenyAssignmentClient {
	client := mgmtauthorization.NewDenyAssignmentsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &denyAssignmentClient{
		DenyAssignmentsClient: client,
	}
}
