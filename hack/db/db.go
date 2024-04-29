package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	DatabaseName        = "DATABASE_NAME"
	DatabaseAccountName = "DATABASE_ACCOUNT_NAME"
	KeyVaultPrefix      = "KEYVAULT_PREFIX"
)

func run(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	_env, err := env.NewCore(ctx, log, env.COMPONENT_TOOLING, cfg)
	if err != nil {
		return err
	}

	tokenCredential, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return err
	}

	if err := _env.ValidateVars(KeyVaultPrefix); err != nil {
		return err
	}
	keyVaultPrefix := _env.GetEnv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	if err := _env.ValidateVars(DatabaseAccountName); err != nil {
		return err
	}

	dbAccountName := _env.GetEnv(DatabaseAccountName)
	clientOptions := &policy.ClientOptions{
		ClientOptions: _env.Environment().ManagedIdentityCredentialOptions().ClientOptions,
	}
	logrusEntry := log.WithField("component", "database")
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, logrusEntry, tokenCredential, clientOptions, _env.SubscriptionID(), _env.ResourceGroup(), dbAccountName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, &noop.Noop{}, aead, dbAccountName)
	if err != nil {
		return err
	}

	dbName, err := DBName(_env)
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
	cfg := viper.GetViper()
	cfg.AutomaticEnv()

	if err := run(context.Background(), log, cfg); err != nil {
		log.Fatal(err)
	}
}

func DBName(_env env.Core) (string, error) {
	if !_env.IsLocalDevelopmentMode() {
		return "ARO", nil
	}

	if err := _env.ValidateVars(DatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return _env.GetEnv(DatabaseName), nil
}
