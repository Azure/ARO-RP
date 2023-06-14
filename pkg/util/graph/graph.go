package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	msgraph_sps "github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
)

// GetServicePrincipalIDByAppID returns a service principal's object ID from
// an application (client) ID.
func GetServicePrincipalIDByAppID(ctx context.Context, graph *msgraph.GraphServiceClient, appId string) (*string, error) {
	filter := fmt.Sprintf("appId eq '%s'", appId)
	requestConfiguration := &msgraph_sps.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_sps.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id"},
		},
	}
	result, err := graph.ServicePrincipals().Get(ctx, requestConfiguration)
	if err != nil {
		return nil, err
	}

	matches := result.GetValue()
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0].GetId(), nil
	// This case should never happen.  A tenant can only have one service principal
	// per application.  This is just to gracefully handle the impossible happening.
	default:
		return nil, fmt.Errorf("%d service principals have appId '%s'", len(matches), appId)
	}
}
