package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

const (
	KeyVaultPrefix = "KEYVAULT_PREFIX"
)

// NewAEADWithCore creates an AEAD encryption manager with resources available
// from the Core env object.
func NewAEADWithCore(ctx context.Context, _env env.Core, encryptionSecretV2Name string, encryptionSecretName string) (AEAD, error) {
	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return nil, fmt.Errorf("MSI KeyVault Authorizer failed with: %s", err.Error())
	}

	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	return NewMulti(
		ctx, serviceKeyvault, encryptionSecretV2Name, encryptionSecretName,
	)
}
