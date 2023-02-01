package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	pkgportal "github.com/Azure/ARO-RP/pkg/portal"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

func portal(ctx context.Context, log *logrus.Entry, audit *logrus.Entry) error {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	if !_env.IsLocalDevelopmentMode() {
		for _, key := range []string{
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
			"PORTAL_HOSTNAME",
		} {
			if _, found := os.LookupEnv(key); !found {
				return fmt.Errorf("environment variable %q unset", key)
			}
		}
	}

	for _, key := range []string{
		"AZURE_PORTAL_CLIENT_ID",
		"AZURE_PORTAL_ACCESS_GROUP_IDS",
		"AZURE_PORTAL_ELEVATED_GROUP_IDS",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	groupIDs, err := parseGroupIDs(os.Getenv("AZURE_PORTAL_ACCESS_GROUP_IDS"))
	if err != nil {
		return err
	}

	elevatedGroupIDs, err := parseGroupIDs(os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"))
	if err != nil {
		return err
	}

	msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	m := statsd.New(ctx, log.WithField("component", "portal"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	g, err := golang.NewMetrics(log.WithField("component", "portal"), m)
	if err != nil {
		return err
	}

	go g.Run()

	// TODO: should not be using the service keyvault here
	serviceKeyvaultURI, err := keyvault.URI(_env, env.ServiceKeyvaultSuffix)
	if err != nil {
		return err
	}

	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, msiAuthorizer)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, m, aead)
	if err != nil {
		return err
	}

	dbOpenShiftClusters, err := database.NewOpenShiftClusters(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	dbPortal, err := database.NewPortal(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	portalKeyvaultURI, err := keyvault.URI(_env, env.PortalKeyvaultSuffix)
	if err != nil {
		return err
	}

	portalKeyvault := keyvault.NewManager(msiKVAuthorizer, portalKeyvaultURI)

	servingKey, servingCerts, err := portalKeyvault.GetCertificateSecret(ctx, env.PortalServerSecretName)
	if err != nil {
		return err
	}

	clientKey, clientCerts, err := portalKeyvault.GetCertificateSecret(ctx, env.PortalServerClientSecretName)
	if err != nil {
		return err
	}

	sessionKey, err := portalKeyvault.GetBase64Secret(ctx, env.PortalServerSessionKeySecretName, "")
	if err != nil {
		return err
	}

	b, err := portalKeyvault.GetBase64Secret(ctx, env.PortalServerSSHKeySecretName, "")
	if err != nil {
		return err
	}

	sshKey, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode())
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

	address := "localhost:8444"
	sshAddress := "localhost:2222"
	if !_env.IsLocalDevelopmentMode() {
		hostname = os.Getenv("PORTAL_HOSTNAME")
		address = ":8444"
		sshAddress = ":2222"
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	sshl, err := net.Listen("tcp", sshAddress)
	if err != nil {
		return err
	}

	log.Printf("listening %s", address)

	p := pkgportal.NewPortal(_env, audit, log.WithField("component", "portal"), log.WithField("component", "portal-access"), l, sshl, verifier, hostname, servingKey, servingCerts, clientID, clientKey, clientCerts, sessionKey, sshKey, groupIDs, elevatedGroupIDs, dbOpenShiftClusters, dbPortal, dialer, m)

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
