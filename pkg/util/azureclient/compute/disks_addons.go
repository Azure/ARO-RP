package compute

import (
	"context"
)

// DisksClientAddons contains addons for DisksClient
type DisksClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, diskName string) error
}

func (c *disksClient) DeleteAndWait(ctx context.Context, resourceGroupName string, diskName string) error {
	future, err := c.Delete(ctx, resourceGroupName, diskName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.DisksClient.Client)
}
