package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// Every change feed this package hands out must be able to make progress past
// a page it cannot read. The generated iterators cannot, and a consumer of one
// which meets such a page stalls for as long as the process runs. This is a
// rule rather than a list, so that a change feed added later is held to it.
func TestEveryChangeFeedIsResilient(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()
	dbc := cosmosdb.NewDatabaseClient(log, nil, &codec.JsonHandle{}, "localhost", nil)

	type pageSkipper interface {
		PagesSkipped() int
	}

	for _, tt := range []struct {
		name       string
		changeFeed func() (any, error)
	}{
		{
			name: "OpenShiftClusters",
			changeFeed: func() (any, error) {
				db, err := NewOpenShiftClusters(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.ChangeFeed(), nil
			},
		},
		{
			name: "Subscriptions",
			changeFeed: func() (any, error) {
				db, err := NewSubscriptions(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.ChangeFeed(), nil
			},
		},
		{
			name: "Gateway",
			changeFeed: func() (any, error) {
				db, err := NewGateway(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.ChangeFeed(), nil
			},
		},
		{
			name: "OpenShiftVersions",
			changeFeed: func() (any, error) {
				db, err := NewOpenShiftVersions(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.ChangeFeed(), nil
			},
		},
		{
			name: "PlatformWorkloadIdentityRoleSets",
			changeFeed: func() (any, error) {
				db, err := NewPlatformWorkloadIdentityRoleSets(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.ChangeFeed(), nil
			},
		},
		{
			// ChangeFeed is not on the MaintenanceManifests interface, and has no
			// consumer today, but is held to the rule all the same.
			name: "MaintenanceManifests",
			changeFeed: func() (any, error) {
				db, err := NewMaintenanceManifests(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.(interface {
					ChangeFeed() cosmosdb.MaintenanceManifestDocumentIterator
				}).ChangeFeed(), nil
			},
		},
		{
			// As for MaintenanceManifests.
			name: "MaintenanceSchedules",
			changeFeed: func() (any, error) {
				db, err := NewMaintenanceSchedules(ctx, dbc, "db")
				if err != nil {
					return nil, err
				}
				return db.(interface {
					ChangeFeed() cosmosdb.MaintenanceScheduleDocumentIterator
				}).ChangeFeed(), nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			iterator, err := tt.changeFeed()
			if err != nil {
				t.Fatal(err)
			}

			if _, ok := iterator.(pageSkipper); !ok {
				t.Errorf("ChangeFeed() returned %T, which cannot skip a page it cannot read", iterator)
			}
		})
	}
}
