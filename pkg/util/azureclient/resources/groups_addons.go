package resources

import (
	"context"
)

type GroupsClientAddons interface {
	Delete(ctx context.Context, resourceGroupName string) (err error)
}

func (c *groupsClient) Delete(ctx context.Context, resourceGroupName string) (err error) {
	future, err := c.GroupsClient.Delete(ctx, resourceGroupName)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.GroupsClient.Client)
}
