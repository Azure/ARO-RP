package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ServicePrincipalClient is a minimal interface for azure ApplicationsClient
type ServicePrincipalClient interface {
	ServicePrincipalClientAddons
	Create(ctx context.Context, parameters azgraphrbac.ServicePrincipalCreateParameters) (result azgraphrbac.ServicePrincipal, err error)
}

type servicePrincipalClient struct {
	azgraphrbac.ServicePrincipalsClient
}

var _ ServicePrincipalClient = &servicePrincipalClient{}

// NewServicePrincipalClient creates a new ServicePrincipalClient
func NewServicePrincipalClient(environment *azureclient.AROEnvironment, tenantID string, authorizer autorest.Authorizer) ServicePrincipalClient {
	client := azgraphrbac.NewServicePrincipalsClientWithBaseURI(environment.GraphEndpoint, tenantID)
	client.Authorizer = authorizer

	return &servicePrincipalClient{
		ServicePrincipalsClient: client,
	}
}
