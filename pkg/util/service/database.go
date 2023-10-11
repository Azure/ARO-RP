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
)

func NewDatabaseClientUsingToken(ctx context.Context, _env env.Core, log *logrus.Entry, m metrics.Emitter, authorizer autorest.Authorizer, aead encryption.AEAD, insecureSkipVerify bool, component string) (cosmosdb.DatabaseClient, error) {
	accountName := os.Getenv(DatabaseAccountName)

	dbc, err := database.NewDatabaseClient(
		log.WithField("component", component),
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
		component,
		m,
		component,
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
