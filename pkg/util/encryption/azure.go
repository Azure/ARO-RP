package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

const (
	KeyVaultPrefix = "KEYVAULT_PREFIX"
)

// NewAEADWithCore creates an AEAD encryption manager with resources available
// from the Core env object.
func NewAEADWithCore(ctx context.Context, _env env.Core, encryptionSecretV2Name string, encryptionSecretName string) (AEAD, error) {
	msiCredential, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault, err := azsecrets.NewClient(serviceKeyvaultURI, msiCredential, _env.Environment().AzureClientOptions())
	if err != nil {
		return nil, fmt.Errorf("cannot create key vault secrets client: %w", err)
	}

	return NewMulti(
		ctx, serviceKeyvault, encryptionSecretV2Name, encryptionSecretName,
	)
}
