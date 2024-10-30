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
)

func getRoleSetsFromEnv() ([]api.PlatformWorkloadIdentityRoleSetProperties, error) {
	const envKey = envPlatformWorkloadIdentityRoleSets
	var roleSets []api.PlatformWorkloadIdentityRoleSetProperties

	// Unmarshal env data into type api.PlatformWorkloadIdentityRoleSet
	return roleSets, getEnvironmentData(envKey, &roleSets)
}

func getPlatformWorkloadIdentityRoleSetDatabase(ctx context.Context, log *logrus.Entry) (database.PlatformWorkloadIdentityRoleSets, error) {
	_env, err := env.NewCore(ctx, log, env.COMPONENT_UPDATE_ROLE_SETS)
	if err != nil {
		return nil, err
	}

	m := statsd.New(ctx, log.WithField("component", "update-role-sets"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, log, m, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating database client: %w", err)
	}

	dbName, err := env.DBName(_env)
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

	for _, doc := range existingRoleSets.PlatformWorkloadIdentityRoleSetDocuments {
		incoming, found := newRoleSets[doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion]
		if found {
			log.Printf("Found Version %q, patching", incoming.OpenShiftVersion)
			_, err := dbPlatformWorkloadIdentityRoleSets.Patch(ctx, doc.ID, func(inFlightDoc *api.PlatformWorkloadIdentityRoleSetDocument) error {
				inFlightDoc.PlatformWorkloadIdentityRoleSet.Properties = incoming
				return nil
			})
			if err != nil {
				return err
			}
			log.Printf("Version %q found", incoming.OpenShiftVersion)
			delete(newRoleSets, incoming.OpenShiftVersion)
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
