package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
)

// ServicePrincipalClient is a minimal interface for azure ApplicationsClient
type ServicePrincipalClient interface {
	ServicePrincipalClientAddons
	Create(ctx context.Context, parameters graphrbac.ServicePrincipalCreateParameters) (result graphrbac.ServicePrincipal, err error)
}

type servicePrincipalClient struct {
	graphrbac.ServicePrincipalsClient
}

var _ ServicePrincipalClient = &servicePrincipalClient{}

// NewServicePrincipalClient creates a new ServicePrincipalClient
func NewServicePrincipalClient(tenantID string, authorizer autorest.Authorizer) ServicePrincipalClient {
	client := graphrbac.NewServicePrincipalsClient(tenantID)
	client.Authorizer = authorizer

	return &servicePrincipalClient{
		ServicePrincipalsClient: client,
	}
}
