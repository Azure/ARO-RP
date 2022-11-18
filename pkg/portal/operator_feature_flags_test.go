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

func TestOperatorFeatureFlags(t *testing.T) {
	const (
		fakeID      = "11111111-2222-2222-2222-333333333333"
		rgName      = "resourcegroupname"
		clusterName = "cluster1"

		fakeKey = "/subscriptions/" + fakeID + "/resourcegroups/" + rgName + "/providers/microsoft.redhatopenshift/openshiftclusters/" + clusterName
	)

	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	_, err := time.Parse(time.RFC3339, "2011-01-02T01:03:00Z")
	if err != nil {
		t.Error(err)
	}

	fixture.AddOpenShiftClusterDocuments(
		&api.OpenShiftClusterDocument{
			ID:  fakeID,
			Key: fakeKey,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: fakeKey,
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: api.OperatorFlags{
						"aro.alertwebhook.enabled":                 "true",
						"aro.autosizednodes.enable":                "false",
						"aro.azuresubnets.enabled":                 "true",
						"aro.azuresubnets.nsg.managed":             "false",
						"aro.azuresubnets.serviceendpoint.managed": "true",
						"aro.banner.enabled":                       "false",
						"aro.checker.enabled":                      "true",
						"aro.dnsmasq.enabled":                      "false",
						"aro.genevalogging.enabled":                "true",
						"aro.imageconfig.enabled":                  "false",
						"aro.machine.enabled":                      "true",
						"aro.machinehealthcheck.enabled":           "false",
						"aro.machinehealthcheck.managed":           "true",
						"aro.machineset.enabled":                   "false",
						"aro.monitoring.enabled":                   "true",
						"aro.nodedrainer.enabled":                  "false",
						"aro.pullsecret.enabled":                   "true",
						"aro.pullsecret.managed":                   "false",
						"aro.rbac.enabled":                         "true",
						"aro.routefix.enabled":                     "false",
						"aro.storageaccounts.enabled":              "true",
						"aro.workaround.enabled":                   "false",
						"rh.srep.muo.enabled":                      "true",
						"rh.srep.muo.managed":                      "false",
					},
				},
			},
		},
	)

	err = fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	req, err := http.NewRequest("GET", "/api/"+fakeID+"/"+rgName+"/"+clusterName+"/featureflags", nil)
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

	var actualOperatorFlags map[string]string
	err = json.NewDecoder(w.Body).Decode(&actualOperatorFlags)
	if err != nil {
		t.Fatal(err)
	}

	expectedOperatorFlags := map[string]string{"aro.alertwebhook.enabled": "true",
		"aro.autosizednodes.enable":                "false",
		"aro.azuresubnets.enabled":                 "true",
		"aro.azuresubnets.nsg.managed":             "false",
		"aro.azuresubnets.serviceendpoint.managed": "true",
		"aro.banner.enabled":                       "false",
		"aro.checker.enabled":                      "true",
		"aro.dnsmasq.enabled":                      "false",
		"aro.genevalogging.enabled":                "true",
		"aro.imageconfig.enabled":                  "false",
		"aro.machine.enabled":                      "true",
		"aro.machinehealthcheck.enabled":           "false",
		"aro.machinehealthcheck.managed":           "true",
		"aro.machineset.enabled":                   "false",
		"aro.monitoring.enabled":                   "true",
		"aro.nodedrainer.enabled":                  "false",
		"aro.pullsecret.enabled":                   "true",
		"aro.pullsecret.managed":                   "false",
		"aro.rbac.enabled":                         "true",
		"aro.routefix.enabled":                     "false",
		"aro.storageaccounts.enabled":              "true",
		"aro.workaround.enabled":                   "false",
		"rh.srep.muo.enabled":                      "true",
		"rh.srep.muo.managed":                      "false",
	}

	res_err := deep.Equal(expectedOperatorFlags, actualOperatorFlags)
	if res_err != nil {
		t.Error(res_err)
	}
}
