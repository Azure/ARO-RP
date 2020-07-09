package insights

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest"
)

// ActivityLogsClient is a minimal interface for azure ActivityLogsClient
type ActivityLogsClient interface {
	ActivityLogsClientAddons
}

type activityLogsClient struct {
	insights.ActivityLogsClient
}

var _ ActivityLogsClient = &activityLogsClient{}

// NewActivityLogsClient creates a new ActivityLogsClient
func NewActivityLogsClient(subscriptionID string, authorizer autorest.Authorizer) ActivityLogsClient {
	client := insights.NewActivityLogsClient(subscriptionID)
	client.Authorizer = authorizer

	return &activityLogsClient{
		ActivityLogsClient: client,
	}
}
