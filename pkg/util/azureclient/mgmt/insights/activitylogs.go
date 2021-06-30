package insights

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights/insights"
)

/* TODO (dranders): Change back to the normal azure-sdk-for-go
once https://github.com/Azure/azure-sdk-for-go/issues/14704 is completed, or the 2015-04-01 azure API version
successfully finds the missing objects:

	activityLog.OperationName.Value == "Microsoft.Compute/virtualMachines/redeploy/action"
	https://github.com/Azure/ARO-RP/blob/924235001fd72793cc9770426dd74821c1e36bce/test/e2e/adminapi_redeployvm.go#L64-L70

Last information on a fix to be provided by Azure stated sometime in 2nd half of calendar year 2021
More details in the github issue, and the referenced ICM (see github issue for icm id)
*/

// ActivityLogsClient is a minimal interface for azure ActivityLogsClient
type ActivityLogsClient interface {
	ActivityLogsClientAddons
}

type activityLogsClient struct {
	insights.ActivityLogsClient
}

var _ ActivityLogsClient = &activityLogsClient{}

// NewActivityLogsClient creates a new ActivityLogsClient
func NewActivityLogsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ActivityLogsClient {
	client := insights.NewActivityLogsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &activityLogsClient{
		ActivityLogsClient: client,
	}
}
