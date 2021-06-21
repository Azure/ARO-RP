package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestClusters(t *testing.T) {
	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		ID:  "00000000-0000-0000-0000-000000000000",
		Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateSucceeded,
			},
		},
	}, &api.OpenShiftClusterDocument{
		ID:  "00000000-0000-0000-0000-000000000001",
		Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/creating",
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/creating",
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateCreating,
			},
		},
	},
		&api.OpenShiftClusterDocument{
			ID:  "00000000-0000-0000-0000-000000000002",
			Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/failedcreate",
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/failedcreate",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateCreating,
				},
			},
		})

	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	w := httptest.NewRecorder()

	p.clusters(w, &http.Request{})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error(w.Header().Get("Content-Type"))
	}

	var r []AdminOpenShiftCluster
	err = json.NewDecoder(w.Body).Decode(&r)
	if err != nil {
		t.Fatal(err)
	}

	expected := []AdminOpenShiftCluster{
		{
			Key:   "00000000-0000-0000-0000-000000000000",
			Name:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
			State: api.ProvisioningStateSucceeded.String(),
		},
		{
			Key:   "00000000-0000-0000-0000-000000000001",
			Name:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/creating",
			State: api.ProvisioningStateCreating.String(),
		},

		{
			Key:         "00000000-0000-0000-0000-000000000002",
			Name:        "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/failedcreate",
			State:       api.ProvisioningStateFailed.String(),
			FailedState: api.ProvisioningStateCreating.String(),
		},
	}

	for _, l := range deep.Equal(expected, r) {
		t.Error(l)
	}
}
