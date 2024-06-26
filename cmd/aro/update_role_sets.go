package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// 1 - Get env data from agent VMs (with getEnvironmentData)
func getRoleSetsFromEnv() ([]api.PlatformWorkloadIdentityRoleSetProperties, error) {
	const envKey = envPlatformWorkloadIdentityRoleSets
	var roleSets []api.PlatformWorkloadIdentityRoleSetProperties

	// Unmarshal env data into type api.PlatformWorkloadIdentityRoleSet
	if err := getEnvironmentData(envKey, &roleSets); err != nil {
		return nil, err
	}

	return roleSets, nil
}

func getPlatformWorkloadIdentityRoleSetDatabase(ctx context.Context, log *logrus.Entry) (database.PlatformWorkloadIdentityRoleSets, error) {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_UPDATE_ROLE_SETS)
	if err != nil {
		return nil, err
	}

	msiToken, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, fmt.Errorf("MSI Authorizer failed with: %s", err.Error())
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(_env.Environment().KeyVaultScope)
	if err != nil {
		return nil, fmt.Errorf("MSI KeyVault Authorizer failed with: %s", err.Error())
	}

	m := statsd.New(ctx, log.WithField("component", "update-role-sets"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	keyVaultPrefix := os.Getenv(envKeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	if err := env.ValidateVars(envDatabaseAccountName); err != nil {
		return nil, err
	}
	dbAccountName := os.Getenv(envDatabaseAccountName)
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

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return nil, err
	}
	dbPlatformWorkloadIdentityRoleSets, err := database.NewPlatformWorkloadIdentityRoleSets(ctx, dbc, dbName)
	if err != nil {
		return nil, err
	}

	return dbPlatformWorkloadIdentityRoleSets, nil
}

func updatePlatformWorkloadIdentityRoleSetsInCosmosDB(ctx context.Context, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets, log *logrus.Entry) error {
	// 2 - Get the existing role set documents, if existing
	// Mostly copied from update_ocp_versions.go
	existingRoleSets, err := dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
		return nil
	}

	incomingRoleSets, err := getRoleSetsFromEnv()
	if err != nil {
		return err
	}

	newRoleSets := make(map[string]api.PlatformWorkloadIdentityRoleSetProperties)
	for _, doc := range incomingRoleSets {
		newRoleSets[doc.OpenShiftVersion] = doc
	}

	// 3 - Put/patch the new role sets to the doc, overwriting whatever is there for that version, or adding if new
	// Mostly copied from update_ocp_versions.go
	for _, doc := range existingRoleSets.PlatformWorkloadIdentityRoleSetDocuments {
		existing, found := newRoleSets[doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion]
		if found {
			log.Printf("Found Version %q, patching", existing.OpenShiftVersion)
			_, err := dbPlatformWorkloadIdentityRoleSets.Patch(ctx, doc.ID, func(inFlightDoc *api.PlatformWorkloadIdentityRoleSetDocument) error {
				inFlightDoc.PlatformWorkloadIdentityRoleSet.Properties = existing
				return nil
			})
			if err != nil {
				return err
			}
			log.Printf("Version %q found", existing.OpenShiftVersion)
			delete(newRoleSets, existing.OpenShiftVersion)
			continue
		}

		log.Printf("Version %q not found, deleting", doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion)
		// Delete via changefeed
		_, err := dbPlatformWorkloadIdentityRoleSets.Patch(ctx, doc.ID,
			func(d *api.PlatformWorkloadIdentityRoleSetDocument) error {
				d.PlatformWorkloadIdentityRoleSet.Deleting = true
				d.TTL = 60
				return nil
			})
		if err != nil {
			return err
		}
	}

	for _, doc := range newRoleSets {
		log.Printf("Version %q not found in database, creating", doc.OpenShiftVersion)
		newDoc := api.PlatformWorkloadIdentityRoleSetDocument{
			ID: dbPlatformWorkloadIdentityRoleSets.NewUUID(),
			PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
				Properties: doc,
			},
		}

		_, err := dbPlatformWorkloadIdentityRoleSets.Create(ctx, &newDoc)
		if err != nil {
			return err
		}
	}

	return nil
}

func updatePlatformWorkloadIdentityRoleSets(ctx context.Context, log *logrus.Entry) error {
	if err := env.ValidateVars("PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"); err != nil {
		return err
	}

	if !env.IsLocalDevelopmentMode() {
		if err := env.ValidateVars("MDM_ACCOUNT", "MDM_NAMESPACE"); err != nil {
			return err
		}
	}

	dbRoleSets, err := getPlatformWorkloadIdentityRoleSetDatabase(ctx, log)
	if err != nil {
		return err
	}

	err = updatePlatformWorkloadIdentityRoleSetsInCosmosDB(ctx, dbRoleSets, log)
	if err != nil {
		return err
	}

	return nil
}
