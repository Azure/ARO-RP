package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/store"
	az "github.com/microsoftgraph/msgraph-sdk-go-core/authentication"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/graph/graphsdk"
)

type GraphServiceClient struct {
	graphsdk.GraphBaseServiceClient
}

// Create a GraphServiceClient for use.
//
// NOTE: If you want to update the underlying SDK, see docs/updating-graphsdk.md for instructions.
func NewGraphServiceClient(adapter abstractions.RequestAdapter) *GraphServiceClient {
	client := graphsdk.NewGraphBaseServiceClient(adapter, store.BackingStoreFactoryInstance)
	return &GraphServiceClient{
		*client,
	}
}

// NewGraphServiceClientWithCredentials instantiates a new GraphServiceClient with provided credentials and scopes
func NewGraphServiceClientWithCredentials(credential azcore.TokenCredential, scopes []string) (*GraphServiceClient, error) {
	if credential == nil {
		return nil, errors.New("credential cannot be nil")
	}

	if len(scopes) == 0 {
		scopes = []string{"https://graph.microsoft.com/.default"}
	}

	validhosts := []string{"graph.microsoft.com", "graph.microsoft.us", "dod-graph.microsoft.us", "graph.microsoft.de", "canary.graph.microsoft.com"}
	auth, err := az.NewAzureIdentityAuthenticationProviderWithScopesAndValidHosts(credential, scopes, validhosts)
	if err != nil {
		return nil, err
	}

	adapter, err := NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, err
	}

	client := NewGraphServiceClient(adapter)
	return client, nil
}

// GetAdapter returns the client current adapter, Method should only be called when the user is certain an adapter has been provided
func (m *GraphServiceClient) GetAdapter() abstractions.RequestAdapter {
	if m.RequestAdapter == nil {
		panic(errors.New("request adapter has not been initialized"))
	}
	return m.RequestAdapter
}
