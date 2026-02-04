package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	pkgportal "github.com/Azure/ARO-RP/pkg/portal"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

func portal(ctx context.Context, _log *logrus.Entry, auditLog *logrus.Entry) error {
	_env, err := env.NewCore(ctx, _log, env.SERVICE_PORTAL)
	if err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		err := env.ValidateVars(
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
			"PORTAL_HOSTNAME")
		if err != nil {
			return err
		}
	}

	err = env.ValidateVars(
		"AZURE_PORTAL_CLIENT_ID",
		"AZURE_PORTAL_ACCESS_GROUP_IDS",
		"AZURE_PORTAL_ELEVATED_GROUP_IDS",
	)
	if err != nil {
		return err
	}

	groupIDs, err := parseGroupIDs(os.Getenv("AZURE_PORTAL_ACCESS_GROUP_IDS"))
	if err != nil {
		return err
	}

	elevatedGroupIDs, err := parseGroupIDs(os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"))
	if err != nil {
		return err
	}

	m := statsd.New(ctx, _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))
	go m.Run(nil)

	g, err := golang.NewMetrics(_env.LoggerForComponent("metrics"), m)
	if err != nil {
		return err
	}

	go g.Run()

	aead, err := encryption.NewAEADWithCore(ctx, _env, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, m, aead)
	if err != nil {
		return err
	}

	dbName, err := env.DBName(_env)
	if err != nil {
		return err
	}

	dbOpenShiftClusters, err := database.NewOpenShiftClusters(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbPortal, err := database.NewPortal(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	dbGroup := database.NewDBGroup().
		WithOpenShiftClusters(dbOpenShiftClusters).
		WithPortal(dbPortal)

	msiCredential, err := _env.NewMSITokenCredential()
	if err != nil {
		return err
	}

	keyVaultPrefix := os.Getenv(encryption.KeyVaultPrefix)
	portalKeyvaultURI := azsecrets.URI(_env, env.PortalKeyvaultSuffix, keyVaultPrefix)
	secretsClient, err := azsecrets.NewClient(portalKeyvaultURI, msiCredential, _env.Environment().AzureClientOptions())
	if err != nil {
		return fmt.Errorf("cannot create key vault secrets client: %w", err)
	}

	serverCertificate, err := secretsClient.GetSecret(ctx, env.PortalServerSecretName, "", nil)
	if err != nil {
		return fmt.Errorf("cannot get server certificate secret: %w", err)
	}

	servingKey, servingCerts, err := azsecrets.ParseSecretAsCertificate(serverCertificate)
	if err != nil {
		return err
	}

	clientCertificate, err := secretsClient.GetSecret(ctx, env.PortalServerClientSecretName, "", nil)
	if err != nil {
		return fmt.Errorf("cannot get client certificate secret: %w", err)
	}

	clientKey, clientCerts, err := azsecrets.ParseSecretAsCertificate(clientCertificate)
	if err != nil {
		return err
	}

	serverSession, err := secretsClient.GetSecret(ctx, env.PortalServerSessionKeySecretName, "", nil)
	if err != nil {
		return fmt.Errorf("cannot get server session secret: %w", err)
	}

	sessionKey, err := azsecrets.ExtractBase64Value(serverSession)
	if err != nil {
		return err
	}

	serverSSHKey, err := secretsClient.GetSecret(ctx, env.PortalServerSSHKeySecretName, "", nil)
	if err != nil {
		return fmt.Errorf("cannot get server ssh key secret: %w", err)
	}

	b, err := azsecrets.ExtractBase64Value(serverSSHKey)
	if err != nil {
		return err
	}

	sshKey, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode(), _env.LoggerForComponent("dialer"))
	if err != nil {
		return err
	}

	clientID := os.Getenv("AZURE_PORTAL_CLIENT_ID")
	verifier, err := oidc.NewVerifier(ctx, _env.Environment().ActiveDirectoryEndpoint+_env.TenantID()+"/v2.0", clientID)
	if err != nil {
		return err
	}

	// In development the portal API is proxied by the frontend dev server which is
	// hosted at localhost:3000, so the hostname needs to be set to that.
	// Set the hostname to localhost:8444 if needing to test compiled portal locally without a frontend dev server
	hostname := "localhost:3000"
	_, noNpm := os.LookupEnv("NO_NPM")
	if noNpm {
		hostname = "localhost:8444"
	}

	address := ":8444"
	sshAddress := ":2222"
	if !_env.IsLocalDevelopmentMode() {
		hostname = os.Getenv("PORTAL_HOSTNAME")
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	sshl, err := net.Listen("tcp", sshAddress)
	if err != nil {
		return err
	}

	_env.Logger().Printf("listening %s", address)

	var size int
	if err := env.ValidateVars(env.OtelAuditQueueSize); err != nil {
		size = 4000
	} else {
		size, err = strconv.Atoi(os.Getenv(env.OtelAuditQueueSize))
		if err != nil {
			return err
		}
	}

	outelAuditClient, err := audit.NewOtelAuditClient(size, _env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	p := pkgportal.NewPortal(_env, auditLog, _env.LoggerForComponent("portal"), _env.LoggerForComponent("portal-access"), outelAuditClient, l, sshl, verifier, hostname, servingKey, servingCerts, clientID, clientKey, clientCerts, sessionKey, sshKey, groupIDs, elevatedGroupIDs, dbGroup, dialer, m)

	return p.Run(ctx)
}

func parseGroupIDs(_groupIDs string) ([]string, error) {
	groupIDs := strings.Split(_groupIDs, ",")
	for _, groupID := range groupIDs {
		_, err := uuid.FromString(groupID)
		if err != nil {
			return nil, err
		}
	}
	return groupIDs, nil
}
