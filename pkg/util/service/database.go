package service

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// NewDatabase creates a CosmosDB database client from the environment configuration.
func NewDatabase(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, withAEAD bool) (cosmosdb.DatabaseClient, error) {
	var aead encryption.AEAD

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	if withAEAD {
		msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
		if err != nil {
			return nil, err
		}

		keyVaultPrefix := os.Getenv(KeyVaultPrefix)
		// TODO: should not be using the service keyvault here
		serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
		serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

		aead, err = encryption.NewMulti(
			ctx,
			serviceKeyvault,
			env.EncryptionSecretV2Name,
			env.EncryptionSecretName,
		)
		if err != nil {
			return nil, err
		}
	}

	dbAccountName := os.Getenv(DatabaseAccountName)
	scope := []string{
		fmt.Sprintf("https://%s.%s", dbAccountName, _env.Environment().CosmosDBDNSSuffixScope),
	}

	logrusEntry := log.WithField("component", "database")

	dbAuthorizer, err := database.NewTokenAuthorizer(
		ctx,
		logrusEntry,
		msiToken,
		dbAccountName,
		scope,
	)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClient(
		logrusEntry,
		_env,
		dbAuthorizer,
		m,
		aead,
		dbAccountName,
	)
	if err != nil {
		return nil, err
	}

	return dbc, nil
}
