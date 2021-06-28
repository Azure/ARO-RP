package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ApplicationsClient is a minimal interface for azure ApplicationsClient
type ApplicationsClient interface {
	ApplicationsClientAddons
	Create(ctx context.Context, parameters azgraphrbac.ApplicationCreateParameters) (result azgraphrbac.Application, err error)
	GetServicePrincipalsIDByAppID(ctx context.Context, applicationID string) (result azgraphrbac.ServicePrincipalObjectResult, err error)
	Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error)
}

type applicationsClient struct {
	azgraphrbac.ApplicationsClient
}

var _ ApplicationsClient = &applicationsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewApplicationsClient(environment *azureclient.AROEnvironment, tenantID string, authorizer autorest.Authorizer) ApplicationsClient {
	client := azgraphrbac.NewApplicationsClientWithBaseURI(environment.GraphEndpoint, tenantID)
	client.Authorizer = authorizer

	return &applicationsClient{
		ApplicationsClient: client,
	}
}
