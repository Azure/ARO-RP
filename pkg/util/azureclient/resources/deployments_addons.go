package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

type DeploymentsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) error
}

func (c *deploymentsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) error {
	future, err := c.DeploymentsClient.CreateOrUpdate(ctx, resourceGroupName, deploymentName, parameters)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.DeploymentsClient.Client)
}
