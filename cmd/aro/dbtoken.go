package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	pkgdbtoken "github.com/Azure/ARO-RP/pkg/dbtoken"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
)

func dbtoken(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_DBTOKEN)
	if err != nil {
		return err
	}

	if err := env.ValidateVars("AZURE_GATEWAY_SERVICE_PRINCIPAL_ID", "AZURE_DBTOKEN_CLIENT_ID"); err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		if err := env.ValidateVars("MDM_ACCOUNT", "MDM_NAMESPACE"); err != nil {
			return err
		}
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "dbtoken"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "dbtoken"), m)
	if err != nil {
		return err
	}

	go g.Run()

	if err := env.ValidateVars(envDatabaseAccountName); err != nil {
		return err
	}

	dbAccountName := os.Getenv(envDatabaseAccountName)

	clientOptions := &policy.ClientOptions{
		ClientOptions: _env.Environment().ManagedIdentityCredentialOptions().ClientOptions,
	}
	logrusEntry := log.WithField("component", "database")
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, logrusEntry, msiToken, clientOptions, _env.SubscriptionID(), _env.ResourceGroup(), dbAccountName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, m, nil, dbAccountName)
	if err != nil {
		return err
	}

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	userc := cosmosdb.NewUserClient(dbc, dbName)

	err = pkgdbtoken.ConfigurePermissions(ctx, dbName, userc)
	if err != nil {
		return err
	}

	if err := env.ValidateVars(envKeyVaultPrefix); err != nil {
		return err
	}
	keyVaultPrefix := os.Getenv(envKeyVaultPrefix)
	dbtokenKeyvaultURI := keyvault.URI(_env, env.DBTokenKeyvaultSuffix, keyVaultPrefix)
	dbtokenKeyvault := keyvault.NewManager(msiKVAuthorizer, dbtokenKeyvaultURI)

	servingKey, servingCerts, err := dbtokenKeyvault.GetCertificateSecret(ctx, env.DBTokenServerSecretName)
	if err != nil {
		return err
	}

	// example value: https://login.microsoftonline.com/11111111-1111-1111-1111-111111111111/v2.0
	issuer := _env.Environment().ActiveDirectoryEndpoint + _env.TenantID() + "/v2.0"
	clientID := os.Getenv("AZURE_DBTOKEN_CLIENT_ID")

	verifier, err := oidc.NewVerifier(ctx, issuer, clientID)
	if err != nil {
		return err
	}

	address := "localhost:8445"
	if !_env.IsLocalDevelopmentMode() {
		address = ":8445"
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Print("listening")

	server, err := pkgdbtoken.NewServer(ctx, _env, log.WithField("component", "dbtoken"), log.WithField("component", "dbtoken-access"), l, servingKey, servingCerts, verifier, userc, m)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}
