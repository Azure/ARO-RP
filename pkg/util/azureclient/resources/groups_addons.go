package resources

import (
	"context"
)

// GroupsClientAddons contains addons for GroupsClient
type GroupsClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string) (err error)
}

func (c *groupsClient) DeleteAndWait(ctx context.Context, resourceGroupName string) error {
	future, err := c.Delete(ctx, resourceGroupName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.GroupsClient.Client)
}
