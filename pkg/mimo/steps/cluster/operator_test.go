package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestResetOperator(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_, log := testlog.New()
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	doc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				OperatorFlags: api.OperatorFlags{
					"foo": "baz",
					"gaz": "data",
				},
				OperatorVersion: "whatever",
			},
		},
	}
	expectedDoc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				OperatorFlags: api.OperatorFlags{
					"foo": "baz",
					"gaz": "data",
				},
				OperatorVersion: "",
			},
		},
	}

	openShiftClustersDatabase, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()
	fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
	fixture.AddOpenShiftClusterDocuments(doc)

	checker := testdatabase.NewChecker()
	checker.AddOpenShiftClusterDocuments(expectedDoc)

	r.NoError(fixture.Create())
	// Should fail checking before we run it
	r.Len(checker.CheckOpenShiftClusters(openShiftClustersClient), 1)

	tc := testtasks.NewFakeTestContext(
		ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
		testtasks.WithOpenShiftDatabase(openShiftClustersDatabase),
		testtasks.WithOpenShiftClusterDocument(doc),
	)

	// Run the step
	r.NoError(ResetOperatorVersion(tc))

	// Passes validation (empty OperatorVersion)
	r.Empty(checker.CheckOpenShiftClusters(openShiftClustersClient))
}
