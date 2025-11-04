package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func IsLocalCosmosDBEnabled() bool {
	envValue := os.Getenv("USE_COSMOS_DB_EMULATOR")
	return envValue == "true" || envValue == "1"
}

// NewDatabaseClient creates a CosmosDB database client from the environment configuration.
func NewDatabaseClientFromEnv(ctx context.Context, _env env.Core, m metrics.Emitter, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	if IsLocalCosmosDBEnabled() {
		return NewLocalDatabaseClient(_env.LoggerForComponent("database"), m, aead)
	}

	dbAccountName, err := env.DBAccountName()
	if err != nil {
		return nil, err
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	scope := []string{
		fmt.Sprintf("https://%s.%s", dbAccountName, _env.Environment().CosmosDBDNSSuffixScope),
	}

	logrusEntry := _env.LoggerForComponent("database")

	dbAuthorizer, err := NewTokenAuthorizer(
		ctx, logrusEntry, msiToken, dbAccountName, scope,
	)
	if err != nil {
		return nil, err
	}

	dbc, err := NewDatabaseClient(
		logrusEntry, _env, dbAuthorizer, m, aead, dbAccountName,
	)
	if err != nil {
		return nil, err
	}

	return dbc, nil
}
