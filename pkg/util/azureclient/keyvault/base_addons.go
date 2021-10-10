package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
)

// BaseClientAddons contains addons for BaseClient
type BaseClientAddons interface {
	GetSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (secrets []azkeyvault.SecretItem, err error)
	GetSecretVersions(ctx context.Context, vaultBaseURL string, secretName string, maxresults *int32) (result []azkeyvault.SecretItem, err error)
}

func (c *baseClient) GetSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (secrets []azkeyvault.SecretItem, err error) {
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

func (c *baseClient) GetSecretVersions(ctx context.Context, vaultBaseURL string, secretName string, maxresults *int32) (secrets []azkeyvault.SecretItem, err error) {
	page, err := c.BaseClient.GetSecretVersions(ctx, vaultBaseURL, secretName, maxresults)
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
