package policyinsights

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mgmtpolicyinsights "github.com/Azure/azure-sdk-for-go/services/preview/policyinsights/mgmt/2020-07-01-preview/policyinsights"
	"github.com/Azure/go-autorest/autorest"
)

// RemediationsClient is a minimal interface for azure RemediationsClient
type RemediationsClient interface {
	CreateOrUpdateAtResourceGroup(ctx context.Context, subscriptionID string, resourceGroupName string, remediationName string, parameters mgmtpolicyinsights.Remediation) (result mgmtpolicyinsights.Remediation, err error)
}

type remediationsClient struct {
	mgmtpolicyinsights.RemediationsClient
}

var _ RemediationsClient = &remediationsClient{}

func NewRemediationsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) RemediationsClient {
	client := mgmtpolicyinsights.NewRemediationsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &remediationsClient{
		RemediationsClient: client,
	}
}
