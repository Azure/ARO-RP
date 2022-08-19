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
	IgnoreList               []string
	MinimumOpenshiftVersions = []api.OpenShiftVersion{
		{
			Version:           "4.11.0",
			ID:                "4.11.0",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:b89ada9261a1b257012469e90d7d4839d0d2f99654f5ce76394fa3f06522b600",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.20",
			ID:                "4.10.20",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:b89ada9261a1b257012469e90d7d4839d0d2f99654f5ce76394fa3f06522b600",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.21",
			ID:                "4.10.21",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:420ee7160d4970304ae97a1b0a77d9bd52af1fd97c597d7cb5d5a2c0d0b72dda",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.22",
			ID:                "4.10.22",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:62c995079672535662ee94ef2358ee6b0e700475c38f6502ca2d3d13d9d7de5b",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.23",
			ID:                "4.10.23",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:e40e49d722cb36a95fa1c03002942b967ccbd7d68de10e003f0baa69abad457b",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.24",
			ID:                "4.10.24",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:aab51636460b5a9757b736a29bc92ada6e6e6282e46b06e6fd483063d590d62a",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.25",
			ID:                "4.10.25",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:ed84fb3fbe026b3bbb4a2637ddd874452ac49c6ead1e15675f257e28664879cc",
			InstallerPullspec: "",
			Enabled:           true,
		},
		{
			Version:           "4.10.26",
			ID:                "4.10.26",
			OpenShiftPullspec: "quay.io/openshift-release-dev/ocp-release@sha256:e1fa1f513068082d97d78be643c369398b0e6820afab708d26acda2262940954",
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
		ignored := false
		version := doc.OpenShiftVersion.ID

		for _, ignoredVer := range IgnoreList {
			if ignoredVer == version {
				ignored = true
				log.Printf("Found Version %s in ignore list, ignoring", doc.ID)
				break
			}
		}

		if !ignored {
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
