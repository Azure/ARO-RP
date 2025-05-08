package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	DatabaseName        = "DATABASE_NAME"
	DatabaseAccountName = "DATABASE_ACCOUNT_NAME"
	KeyVaultPrefix      = "KEYVAULT_PREFIX"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	_env, err := env.NewCore(ctx, log, env.COMPONENT_TOOLING)
	if err != nil {
		return err
	}

	msiCredential, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}

	if err := env.ValidateVars(KeyVaultPrefix); err != nil {
		return err
	}
	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := azsecrets.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault, err := azsecrets.NewClient(serviceKeyvaultURI, msiCredential, _env.Environment().AzureClientOptions())
	if err != nil {
		return fmt.Errorf("cannot create key vault secrets client: %w", err)
	}

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	if err := env.ValidateVars(DatabaseAccountName); err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, log, &noop.Noop{}, aead)
	if err != nil {
		return err
	}

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	openShiftClusters, err := database.NewOpenShiftClusters(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	doc, err := openShiftClusters.Get(ctx, strings.ToLower(os.Args[1]))
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(doc)
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}

func DBName(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
		return "ARO", nil
	}

	if err := env.ValidateVars(DatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(DatabaseName), nil
}
