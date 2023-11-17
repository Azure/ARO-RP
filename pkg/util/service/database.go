package service

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	pkgdbtoken "github.com/Azure/ARO-RP/pkg/dbtoken"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

func NewDatabaseClientUsingToken(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, authorizer autorest.Authorizer, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	accountName := os.Getenv(DatabaseAccountName)
	insecureSkipVerify := _env.IsLocalDevelopmentMode()

	dbc, err := database.NewDatabaseClient(
		log.WithField("component", "database"),
		_env,
		nil,
		m,
		aead,
		accountName,
	)
	if err != nil {
		return nil, err
	}

	dbTokenURL, err := GetDBTokenURL(_env.IsLocalDevelopmentMode())
	if err != nil {
		return nil, err
	}
	dbRefresher := pkgdbtoken.NewRefresher(
		log,
		_env,
		authorizer,
		insecureSkipVerify,
		dbc,
		m,
		dbTokenURL,
	)

	go func() {
		_ = dbRefresher.Run(ctx)
	}()

	log.Print("waiting for database token")
	for !dbRefresher.HasSyncedOnce() {
		time.Sleep(time.Second)
	}

	return dbc, nil
}

func NewDatabaseClientUsingMasterKey(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, authorizer autorest.Authorizer, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	dbAccountName := os.Getenv(DatabaseAccountName)
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, authorizer, dbAccountName)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClient(
		log.WithField("component", "database"),
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

func NewDatabase(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, dbPreference DB_TYPE, withAEAD bool) (cosmosdb.DatabaseClient, error) {
	var aead encryption.AEAD

	if withAEAD {
		msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
		if err != nil {
			return nil, err
		}

		keyVaultPrefix := os.Getenv(KeyVaultPrefix)
		// TODO: should not be using the service keyvault here
		serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
		serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

		aead, err = encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
		if err != nil {
			return nil, err
		}
	}

	if dbPreference == DB_ALWAYS_DBTOKEN || (dbPreference == DB_DBTOKEN_PROD_MASTERKEY_DEV && _env.IsLocalDevelopmentMode()) {
		msiAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().ResourceManagerScope)
		if err != nil {
			return nil, err
		}

		return NewDatabaseClientUsingMasterKey(ctx, _env, log, m, msiAuthorizer, aead)
	}

	// Access token GET request needs to be:
	// http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=$AZURE_DBTOKEN_CLIENT_ID
	//
	// In this context, the "resource" parameter is passed to azidentity as a
	// "scope" argument even though a scope normally consists of an endpoint URL.
	scope := os.Getenv("AZURE_" + _env.Component() + "_CLIENT_ID")
	msiRefresherAuthorizer, err := _env.NewMSIAuthorizer(scope)
	if err != nil {
		return nil, err
	}

	return NewDatabaseClientUsingToken(ctx, _env, log, m, msiRefresherAuthorizer, aead)
}
