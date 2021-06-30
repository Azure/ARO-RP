package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	pkgdbtoken "github.com/Azure/ARO-RP/pkg/dbtoken"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
)

func dbtoken(ctx context.Context, log *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		for _, key := range []string{
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		} {
			if _, found := os.LookupEnv(key); !found {
				return fmt.Errorf("environment variable %q unset", key)
			}
		}
	}

	msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "dbtoken"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"))

	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, msiAuthorizer)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, m, nil)
	if err != nil {
		return err
	}

	dbid, err := database.Name(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	userc := cosmosdb.NewUserClient(dbc, dbid)

	err = pkgdbtoken.ConfigurePermissions(ctx, dbid, userc)
	if err != nil {
		return err
	}

	dbtokenKeyvaultURI, err := keyvault.URI(_env, env.DBTokenKeyvaultSuffix)
	if err != nil {
		return err
	}

	dbtokenKeyvault := keyvault.NewManager(msiKVAuthorizer, dbtokenKeyvaultURI)

	servingKey, servingCerts, err := dbtokenKeyvault.GetCertificateSecret(ctx, env.DBTokenServerSecretName)
	if err != nil {
		return err
	}

	// example value: https://login.microsoftonline.com/11111111-1111-1111-1111-111111111111/
	issuer := _env.Environment().ActiveDirectoryEndpoint + _env.TenantID() + "/"

	// example value: https://dbtoken.aro.azure.com/
	clientID := "https://dbtoken." + _env.Environment().AppSuffix + "/"

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

	server, err := pkgdbtoken.NewServer(ctx, _env, log.WithField("component", "dbtoken"), log.WithField("component", "dbtoken-access"), l, servingKey, servingCerts, verifier, userc)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}
