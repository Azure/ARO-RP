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

// NewAEADWithCore creates an AEAD encryption manager with resources available
// from the Core env object.
func NewAEADWithCore(ctx context.Context, _env env.Core, encryptionSecretV2Name string, encryptionSecretName string) (encryption.AEAD, error) {
	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return nil, err
	}

	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	return encryption.NewMulti(
		ctx, serviceKeyvault, encryptionSecretV2Name, encryptionSecretName,
	)
}

// NewDatabaseClient creates a CosmosDB database client from the environment configuration.
func NewDatabaseClient(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	if err := env.ValidateVars(DatabaseAccountName); err != nil {
		return nil, err
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	dbAccountName := os.Getenv(DatabaseAccountName)
	scope := []string{
		fmt.Sprintf("https://%s.%s", dbAccountName, _env.Environment().CosmosDBDNSSuffixScope),
	}

	logrusEntry := log.WithField("component", "database")

	dbAuthorizer, err := database.NewTokenAuthorizer(
		ctx, logrusEntry, msiToken, dbAccountName, scope,
	)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClient(
		logrusEntry, _env, dbAuthorizer, m, aead, dbAccountName,
	)
	if err != nil {
		return nil, err
	}

	return dbc, nil
}
