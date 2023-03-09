package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func getLatestOCPVersions(ctx context.Context, log *logrus.Entry) ([]api.OpenShiftVersion, error) {
	env, err := env.NewCoreForCI(ctx, log)
	if err != nil {
		return nil, err
	}
	dstAcr := os.Getenv("DST_ACR_NAME")
	acrDomainSuffix := "." + env.Environment().ContainerRegistryDNSSuffix

	dstRepo := dstAcr + acrDomainSuffix
	ocpVersions := []api.OpenShiftVersion{}

	for _, vers := range version.HiveInstallStreams {
		ocpVersions = append(ocpVersions, api.OpenShiftVersion{
			Properties: api.OpenShiftVersionProperties{
				Version:           vers.Version.String(),
				OpenShiftPullspec: vers.PullSpec,
				InstallerPullspec: fmt.Sprintf("%s/aro-installer:release-%s", dstRepo, vers.Version.MinorVersion()),
				Enabled:           true,
			},
		})
	}

	return ocpVersions, nil
}

func getVersionsDatabase(ctx context.Context, log *logrus.Entry) (database.OpenShiftVersions, error) {
	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return nil, err
	}

	for _, key := range []string{
		"DST_ACR_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	if !_env.IsLocalDevelopmentMode() {
		for _, key := range []string{
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset", key)
			}
		}
	}

	msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("MSI Authorizer failed with: %s", err.Error())
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, fmt.Errorf("MSI KeyVault Authorizer failed with: %s", err.Error())
	}

	m := statsd.New(ctx, log.WithField("component", "update-ocp-versions"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	serviceKeyvaultURI, err := keyvault.URI(_env, env.ServiceKeyvaultSuffix)
	if err != nil {
		return nil, err
	}

	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, msiAuthorizer)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, m, aead)
	if err != nil {
		return nil, err
	}

	dbOpenShiftVersions, err := database.NewOpenShiftVersions(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return nil, err
	}

	return dbOpenShiftVersions, nil
}

func updateOpenShiftVersions(ctx context.Context, dbOpenShiftVersions database.OpenShiftVersions, log *logrus.Entry) error {
	existingVersions, err := dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return err
	}

	latestVersions, err := getLatestOCPVersions(ctx, log)
	if err != nil {
		return err
	}

	newVersions := make(map[string]api.OpenShiftVersion)
	for _, doc := range latestVersions {
		newVersions[doc.Properties.Version] = doc
	}

	for _, doc := range existingVersions.OpenShiftVersionDocuments {
		existing, found := newVersions[doc.OpenShiftVersion.Properties.Version]
		if found {
			log.Printf("Found Version %q, patching", existing.Properties.Version)
			_, err := dbOpenShiftVersions.Patch(ctx, doc.ID, func(inFlightDoc *api.OpenShiftVersionDocument) error {
				inFlightDoc.OpenShiftVersion = &existing
				return nil
			})
			if err != nil {
				return err
			}
			log.Printf("Version %q found", existing.Properties.Version)
			delete(newVersions, existing.Properties.Version)
			continue
		}

		log.Printf("Version %q not found, deleting", doc.OpenShiftVersion.Properties.Version)
		err := dbOpenShiftVersions.Delete(ctx, doc)
		if err != nil {
			return err
		}
	}

	for _, doc := range newVersions {
		log.Printf("Version %q not found in database, creating", doc.Properties.Version)
		newDoc := api.OpenShiftVersionDocument{
			ID:               dbOpenShiftVersions.NewUUID(),
			OpenShiftVersion: &doc,
		}
		_, err := dbOpenShiftVersions.Create(ctx, &newDoc)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateOCPVersions(ctx context.Context, log *logrus.Entry) error {
	dbOpenShiftVersions, err := getVersionsDatabase(ctx, log)
	if err != nil {
		return err
	}

	err = updateOpenShiftVersions(ctx, dbOpenShiftVersions, log)
	if err != nil {
		return err
	}
	return nil
}
