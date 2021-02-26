package insights

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// ActivityLogsClient is a minimal interface for azure ActivityLogsClient
type ActivityLogsClient interface {
	ActivityLogsClientAddons
}

type activityLogsClient struct {
	mgmtinsights.ActivityLogsClient
}

var _ ActivityLogsClient = &activityLogsClient{}

// NewActivityLogsClient creates a new ActivityLogsClient
func NewActivityLogsClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) ActivityLogsClient {
	client := mgmtinsights.NewActivityLogsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &activityLogsClient{
		ActivityLogsClient: client,
	}
}
