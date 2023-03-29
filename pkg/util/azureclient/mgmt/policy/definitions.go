package policy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtpolicy "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-09-01/policy"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// DefinitionsClient is a minimal interface for azure DefinitionsClient
type DefinitionsClient interface {
	CreateOrUpdate(ctx context.Context, policyDefinitionName string, parameters mgmtpolicy.Definition) (results mgmtpolicy.Definition, err error)
	Delete(ctx context.Context, policyDefinitionName string) (result autorest.Response, err error)
}

type definitionsClient struct {
	mgmtpolicy.DefinitionsClient
}

var _ DefinitionsClient = &definitionsClient{}

func NewDefinitionsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DefinitionsClient {
	client := mgmtpolicy.NewDefinitionsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &definitionsClient{
		DefinitionsClient: client,
	}
}
