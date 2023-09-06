package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	core "github.com/microsoftgraph/msgraph-sdk-go-core"
)

var clientOptions = core.GraphClientOptions{
	GraphServiceVersion:        "", //v1 doesn't include the service version in the telemetry header
	GraphServiceLibraryVersion: "1.15.0",
}

// GraphRequestAdapter is the core service used by GraphBaseServiceClient to make requests to Microsoft Graph.
type GraphRequestAdapter struct {
	core.GraphRequestAdapterBase
}

// NewGraphRequestAdapter creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapter(authenticationProvider absauth.AuthenticationProvider) (*GraphRequestAdapter, error) {
	baseAdapter, err := core.NewGraphRequestAdapterBaseWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, clientOptions, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	result := &GraphRequestAdapter{
		GraphRequestAdapterBase: *baseAdapter,
	}

	return result, nil
}
