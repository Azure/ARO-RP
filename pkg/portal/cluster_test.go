package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestClusterList(t *testing.T) {
	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	parsedTime, err := time.Parse(time.RFC3339, "2011-01-02T01:03:00Z")
	if err != nil {
		t.Error(err)
	}

	fixture.AddOpenShiftClusterDocuments(
		&api.OpenShiftClusterDocument{
			ID:  "00000000-0000-0000-0000-000000000000",
			Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState: api.ProvisioningStateSucceeded,
					CreatedAt:         parsedTime,
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

	err = fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	req, err := http.NewRequest("GET", "/api/clusters", nil)
	if err != nil {
		t.Error(err)
	}

	aadAuthenticatedRouter := mux.NewRouter()
	p.aadAuthenticatedRoutes(aadAuthenticatedRouter, nil, nil, nil)
	w := httptest.NewRecorder()
	aadAuthenticatedRouter.ServeHTTP(w, req)

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
			Key:               "00000000-0000-0000-0000-000000000000",
			ResourceId:        "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
			Name:              "succeeded",
			ResourceGroup:     "resourceGroupName",
			Subscription:      "00000000-0000-0000-0000-000000000000",
			CreatedAt:         "2011-01-02T01:03:00Z",
			LastModified:      "Unknown",
			ProvisioningState: api.ProvisioningStateSucceeded.String(),
		},
		{
			Key:               "00000000-0000-0000-0000-000000000001",
			ResourceId:        "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/creating",
			Name:              "creating",
			ResourceGroup:     "resourceGroupName",
			Subscription:      "00000000-0000-0000-0000-000000000000",
			CreatedAt:         "Unknown",
			LastModified:      "Unknown",
			ProvisioningState: api.ProvisioningStateCreating.String(),
		},

		{
			Key:                     "00000000-0000-0000-0000-000000000002",
			ResourceId:              "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/failedcreate",
			Name:                    "failedcreate",
			ResourceGroup:           "resourceGroupName",
			Subscription:            "00000000-0000-0000-0000-000000000000",
			CreatedAt:               "Unknown",
			LastModified:            "Unknown",
			ProvisioningState:       api.ProvisioningStateFailed.String(),
			FailedProvisioningState: api.ProvisioningStateCreating.String(),
		},
	}

	for _, l := range deep.Equal(expected, r) {
		t.Error(l)
	}
}

func TestClusterDetail(t *testing.T) {
	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	parsedTime, err := time.Parse(time.RFC3339, "2011-01-02T01:03:00Z")
	if err != nil {
		t.Error(err)
	}

	fixture.AddOpenShiftClusterDocuments(
		&api.OpenShiftClusterDocument{
			ID:  "00000000-0000-0000-0000-000000000000",
			Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
				Properties: api.OpenShiftClusterProperties{
					LastProvisioningState: api.ProvisioningStateCreating,
					ProvisioningState:     api.ProvisioningStateSucceeded,
					CreatedAt:             parsedTime,
					ProvisionedBy:         "aaaaab",
					InfraID:               "blah",
					ArchitectureVersion:   2,
					CreatedBy:             "aaa",
					ClusterProfile: api.ClusterProfile{
						Version: "4.4.4",
					},
					APIServerProfile: api.APIServerProfile{
						IP:         "1.2.3.4",
						IntIP:      "2.3.4.5",
						Visibility: api.VisibilityPrivate,
						URL:        "example.com",
					},
				},
				SystemData: api.SystemData{
					LastModifiedAt: &parsedTime,
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

	err = fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	req, err := http.NewRequest("GET", "/api/00000000-0000-0000-0000-000000000000/resourcegroupname/succeeded", nil)
	if err != nil {
		t.Error(err)
	}

	aadAuthenticatedRouter := mux.NewRouter()
	p.aadAuthenticatedRoutes(aadAuthenticatedRouter, nil, nil, nil)
	w := httptest.NewRecorder()
	aadAuthenticatedRouter.ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error(w.Body)
		t.Error(w.Header().Get("Content-Type"))
	}

	var r map[string]string
	err = json.NewDecoder(w.Body).Decode(&r)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"resourceId":              "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/succeeded",
		"name":                    "succeeded",
		"resourceGroup":           "resourcegroupname",
		"subscription":            "00000000-0000-0000-0000-000000000000",
		"createdAt":               "2011-01-02T01:03:00Z",
		"infraId":                 "blah",
		"version":                 "4.4.4",
		"createdBy":               "aaa",
		"provisionedBy":           "aaaaab",
		"architectureVersion":     "2",
		"failedProvisioningState": "",
		"apiServerVisibility":     "Private",
		"lastAdminUpdateError":    "",
		"lastProvisioningState":   api.ProvisioningStateCreating.String(),
		"provisioningState":       api.ProvisioningStateSucceeded.String(),
		"installStatus":           "Installed",
	}

	for _, l := range deep.Equal(expected, r) {
		t.Error(l)
	}
}
