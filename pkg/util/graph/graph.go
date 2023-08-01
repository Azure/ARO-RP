package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	msgraph_aps "github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
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

// GetServicePrincipalExpiryByAppID returns a service principal's expiry date from
// an application (client) ID.
func GetServicePrincipalExpiryByAppID(ctx context.Context, graph *msgraph.GraphServiceClient, appId string) (*time.Time, error) {
	filter := fmt.Sprintf("appId eq '%s'", appId)
	requestConfiguration := &msgraph_aps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_aps.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"passwordCredentials"},
		},
	}

	result, err := graph.Applications().Get(ctx, requestConfiguration)
	if err != nil {
		return nil, err
	}

	var endDateTime *time.Time
	matches := result.GetValue()

	for _, element := range matches[0].GetPasswordCredentials() {
		e := element.(*models.PasswordCredential)
		endDateTime = e.GetEndDateTime()
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return endDateTime, nil
	default:
		return nil, fmt.Errorf("%d SP expiry dates for appId '%s'", len(matches), appId)
	}
}
