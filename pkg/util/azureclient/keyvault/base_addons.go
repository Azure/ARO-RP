package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
)

// BaseClientAddons contains addons for BaseClient
type BaseClientAddons interface {
	GetSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (secrets []keyvault.SecretItem, err error)
}

func (c *baseClient) GetSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (secrets []keyvault.SecretItem, err error) {
	page, err := c.BaseClient.GetSecrets(ctx, vaultBaseURL, maxresults)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		secrets = append(secrets, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return secrets, nil
}
