package armpolicy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type DefinitionsClient interface {
	CreateOrUpdate(ctx context.Context, policyDefinitionName string, parameters armpolicy.Definition, options *armpolicy.DefinitionsClientCreateOrUpdateOptions) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, policyDefinitionName string, options *armpolicy.DefinitionsClientDeleteOptions) (armpolicy.DefinitionsClientDeleteResponse, error)
}

type definitionsClient struct {
	*armpolicy.DefinitionsClient
}

func NewDefinitionsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (DefinitionsClient, error) {
	clientFactory, err := armpolicy.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &definitionsClient{clientFactory.NewDefinitionsClient()}, nil
}
