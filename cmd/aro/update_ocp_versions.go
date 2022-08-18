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
)

var (
	MinimumOpenshiftVersions = []api.OpenShiftVersion{
		{
			ID:                "4.12",
			Version:           "4.12.0",
			OpenShiftPullspec: "",
			InstallerPullspec: "",
			Enabled:           false,
		},
		{
			ID:                "4.11",
			Version:           "4.11.0",
			OpenShiftPullspec: "",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			ID:                "4.10",
			Version:           "4.10.0",
			OpenShiftPullspec: "",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			ID:                "4.9",
			Version:           "4.9.0",
			OpenShiftPullspec: "",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			ID:                "4.8",
			Version:           "4.8.0",
			OpenShiftPullspec: "",
			InstallerPullspec: "",
			Enabled:           true,
		},
	}
)

func updateOCPVersions(ctx context.Context, log *logrus.Entry) error {

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

	m := statsd.New(ctx, log.WithField("component", "update-ocp-versions"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

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

	dbOpenShiftVersions, err := database.NewOpenShiftVersions(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	versions, err := dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return err
	}

	newVersions := make(map[string]api.OpenShiftVersion)
	for _, doc := range MinimumOpenshiftVersions {
		newVersions[doc.ID] = doc
	}

	for _, doc := range versions.OpenShiftVersionDocuments {
		foundDoc := false
		version := doc.OpenShiftVersion.ID
		for _, minVer := range MinimumOpenshiftVersions {
			if version == minVer.ID {
				foundDoc = true
				log.Printf("Found Version %s in min version list, patching", doc.ID)
				_, err := dbOpenShiftVersions.Patch(ctx, "1", func(inFlightDoc *api.OpenShiftVersionDocument) error {
					inFlightDoc.OpenShiftVersion = &minVer
					return nil
				})
				if err != nil {
					return err
				}
				log.Printf("Version %s found", minVer.ID)
				delete(newVersions, minVer.ID)
			}
		}
		if !foundDoc {
			log.Printf("Version %s not found, deleting", doc.ID)
			err := dbOpenShiftVersions.Delete(ctx, doc)
			if err != nil {
				return err
			}
		}
	}

	for _, doc := range newVersions {
		log.Printf("Version %s not found in database, creating", doc.ID)
		newDoc := api.OpenShiftVersionDocument{
			ID:               doc.ID,
			OpenShiftVersion: &doc,
		}
		_, err := dbOpenShiftVersions.Create(ctx, &newDoc)
		if err != nil {
			return err
		}
	}

	return nil
}
