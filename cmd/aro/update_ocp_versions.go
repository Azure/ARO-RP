package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Corresponds to configuration.openShiftVersions in RP-Config
type OpenShiftVersions struct {
	DefaultStream  map[string]string
	InstallStreams map[string]string
}

func getEnvironmentData(env env.Core, envKey string, envData any) error {
	var err error

	jsonData := []byte(env.GetEnv(envKey))

	// For Azure DevOps pipelines, the JSON data is Base64-encoded
	// since it's embedded in JSON-formatted build artifacts.  But
	// let's not force that on local development mode.
	if !env.IsLocalDevelopmentMode() {
		jsonData, err = base64.StdEncoding.DecodeString(string(jsonData))
		if err != nil {
			return fmt.Errorf("%s: Failed to decode base64: %w", envKey, err)
		}
	}

	if err = json.Unmarshal(jsonData, envData); err != nil {
		return fmt.Errorf("%s: Failed to parse JSON: %w", envKey, err)
	}

	return nil
}

func getOpenShiftVersions(env env.Core) (*OpenShiftVersions, error) {
	const envKey = envOpenShiftVersions
	var openShiftVersions OpenShiftVersions

	if err := getEnvironmentData(env, envKey, &openShiftVersions); err != nil {
		return nil, err
	}

	// The DefaultStream map must have exactly one entry.
	numDefaultStreams := len(openShiftVersions.DefaultStream)
	if numDefaultStreams != 1 {
		return nil, fmt.Errorf("%s: DefaultStream must have exactly 1 entry, found %d", envKey, numDefaultStreams)
	}

	return &openShiftVersions, nil
}

func getInstallerImageDigests(env env.Core) (map[string]string, error) {
	// INSTALLER_IMAGE_DIGESTS is the mapping of a minor version to
	// the aro-installer wrapper digest.  This allows us to utilize
	// Azure Safe Deployment Practices (SDP) instead of pushing the
	// version tag and deploying to all regions at once.
	const envKey = envInstallerImageDigests
	var installerImageDigests map[string]string

	if err := getEnvironmentData(env, envKey, &installerImageDigests); err != nil {
		return nil, err
	}

	return installerImageDigests, nil
}

func appendOpenShiftVersions(ocpVersions []api.OpenShiftVersion, installStreams map[string]string, installerImageName string, installerImageDigests map[string]string, isDefault bool) ([]api.OpenShiftVersion, error) {
	for fullVersion, openShiftPullspec := range installStreams {
		openShiftVersion, err := version.ParseVersion(fullVersion)
		if err != nil {
			return nil, err
		}
		fullVersion = openShiftVersion.String() // trimmed of whitespace
		minorVersion := openShiftVersion.MinorVersion()
		installerDigest, ok := installerImageDigests[minorVersion]
		if !ok {
			return nil, fmt.Errorf("no installer digest for version %s", minorVersion)
		}
		installerPullspec := fmt.Sprintf("%s:%s@%s", installerImageName, minorVersion, installerDigest)

		ocpVersions = append(ocpVersions, api.OpenShiftVersion{
			Properties: api.OpenShiftVersionProperties{
				Version:           fullVersion,
				OpenShiftPullspec: openShiftPullspec,
				InstallerPullspec: installerPullspec,
				Enabled:           true,
				Default:           isDefault,
			},
		})
	}

	return ocpVersions, nil
}

func getLatestOCPVersions(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) ([]api.OpenShiftVersion, error) {
	env, err := env.NewCoreForCI(ctx, log, cfg)
	if err != nil {
		return nil, err
	}
	dstAcr := env.GetEnv("DST_ACR_NAME")
	acrDomainSuffix := "." + env.Environment().ContainerRegistryDNSSuffix
	installerImageName := dstAcr + acrDomainSuffix + "/aro-installer"

	openShiftVersions, err := getOpenShiftVersions(env)
	if err != nil {
		return nil, err
	}

	installerImageDigests, err := getInstallerImageDigests(env)
	if err != nil {
		return nil, err
	}

	ocpVersions := make([]api.OpenShiftVersion, 0, len(openShiftVersions.DefaultStream)+len(openShiftVersions.InstallStreams))

	ocpVersions, err = appendOpenShiftVersions(ocpVersions, openShiftVersions.DefaultStream, installerImageName, installerImageDigests, true)
	if err != nil {
		return nil, err
	}

	ocpVersions, err = appendOpenShiftVersions(ocpVersions, openShiftVersions.InstallStreams, installerImageName, installerImageDigests, false)
	if err != nil {
		return nil, err
	}

	return ocpVersions, nil
}

func getVersionsDatabase(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) (database.OpenShiftVersions, error) {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_UPDATE_OCP_VERSIONS, cfg)
	if err != nil {
		return nil, err
	}

	if err = _env.ValidateVars("DST_ACR_NAME"); err != nil {
		return nil, err
	}

	if !_env.IsLocalDevelopmentMode() {
		if err = _env.ValidateVars("MDM_ACCOUNT", "MDM_NAMESPACE"); err != nil {
			return nil, err
		}
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, fmt.Errorf("MSI Authorizer failed with: %s", err.Error())
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return nil, fmt.Errorf("MSI KeyVault Authorizer failed with: %s", err.Error())
	}

	m := statsd.New(ctx, log.WithField("component", "update-ocp-versions"), _env, _env.GetEnv("MDM_ACCOUNT"), _env.GetEnv("MDM_NAMESPACE"), _env.GetEnv("MDM_STATSD_SOCKET"))

	if err := _env.ValidateVars(envKeyVaultPrefix); err != nil {
		return nil, err
	}
	keyVaultPrefix := _env.GetEnv(envKeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	if err := _env.ValidateVars(envDatabaseAccountName); err != nil {
		return nil, err
	}

	dbAccountName := _env.GetEnv(envDatabaseAccountName)
	clientOptions := &policy.ClientOptions{
		ClientOptions: _env.Environment().ManagedIdentityCredentialOptions().ClientOptions,
	}
	logrusEntry := log.WithField("component", "database")
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, logrusEntry, msiToken, clientOptions, _env.SubscriptionID(), _env.ResourceGroup(), dbAccountName)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClient(log.WithField("component", "database"), _env, dbAuthorizer, m, aead, dbAccountName)
	if err != nil {
		return nil, err
	}

	dbName, err := DBName(_env)
	if err != nil {
		return nil, err
	}
	dbOpenShiftVersions, err := database.NewOpenShiftVersions(ctx, dbc, dbName)
	if err != nil {
		return nil, err
	}

	return dbOpenShiftVersions, nil
}

func updateOpenShiftVersions(ctx context.Context, dbOpenShiftVersions database.OpenShiftVersions, log *logrus.Entry, cfg *viper.Viper) error {
	existingVersions, err := dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return err
	}

	latestVersions, err := getLatestOCPVersions(ctx, log, cfg)
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
		// Delete via changefeed
		_, err := dbOpenShiftVersions.Patch(ctx, doc.ID,
			func(d *api.OpenShiftVersionDocument) error {
				d.OpenShiftVersion.Deleting = true
				d.TTL = 60
				return nil
			})
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

func updateOCPVersions(ctx context.Context, log *logrus.Entry, cfg *viper.Viper) error {
	dbOpenShiftVersions, err := getVersionsDatabase(ctx, log, cfg)
	if err != nil {
		return err
	}

	err = updateOpenShiftVersions(ctx, dbOpenShiftVersions, log, cfg)
	if err != nil {
		return err
	}
	return nil
}
