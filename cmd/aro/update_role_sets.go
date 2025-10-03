package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func getRoleSetsFromEnv() ([]api.PlatformWorkloadIdentityRoleSetProperties, error) {
	const envKey = envPlatformWorkloadIdentityRoleSets
	var roleSets []api.PlatformWorkloadIdentityRoleSetProperties

	// Unmarshal env data into type api.PlatformWorkloadIdentityRoleSet
	return roleSets, getEnvironmentData(envKey, &roleSets)
}

func getPlatformWorkloadIdentityRoleSetDatabase(ctx context.Context, _log *logrus.Entry) (database.PlatformWorkloadIdentityRoleSets, error) {
	_env, err := env.NewCore(ctx, _log, env.SERVICE_UPDATE_ROLE_SETS)
	if err != nil {
		return nil, err
	}

	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, &noop.Noop{}, nil)
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

	// check if 4.16 is present in the incoming payload
	if doc, ok := newRoleSets["4.16"]; ok {
		log.Infof("incoming payload contains %s", doc.OpenShiftVersion)
	} else {
		log.Infof("incoming payload does not contain 4.16")
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
				Name:       doc.OpenShiftVersion,
				Type:       api.PlatformWorkloadIdentityRoleSetsType,
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
