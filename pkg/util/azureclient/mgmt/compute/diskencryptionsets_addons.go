package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

// DiskEncryptionSetsAddons contains addons for ResourceSkusClient
type DiskEncryptionSetsClientAddons interface {
	DeleteAndWait(ctx context.Context, resourceGroupName string, diskEncryptionSetName string) error
}

func (c *diskEncryptionSetsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, diskEncryptionSetName string) error {
	future, err := c.Delete(ctx, resourceGroupName, diskEncryptionSetName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
