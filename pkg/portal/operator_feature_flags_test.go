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

		fakeID_0 = "00000000-0000-0000-0000-000000000000"
		fakeID_2 = "00000000-0000-0000-0000-000000000001"
		fakeID_3 = "00000000-0000-0000-0000-000000000002"

		fakeKey_2 = "/subscriptions/" + fakeID_0 + "/resourcegroups/" + rgName + "/providers/microsoft.redhatopenshift/openshiftclusters/cluster2"
		fakeKey_3 = "/subscriptions/" + fakeID_0 + "/resourcegroups/" + rgName + "/providers/microsoft.redhatopenshift/openshiftclusters/cluster3"
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
		}, &api.OpenShiftClusterDocument{
			ID:  fakeID_2,
			Key: fakeKey_2,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: fakeKey_2,
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: api.OperatorFlags{
						"aro.alertwebhook.enabled":                 "false",
						"aro.autosizednodes.enable":                "true",
						"aro.azuresubnets.enabled":                 "false",
						"aro.azuresubnets.nsg.managed":             "true",
						"aro.azuresubnets.serviceendpoint.managed": "false",
						"aro.banner.enabled":                       "true",
						"aro.checker.enabled":                      "false",
						"aro.dnsmasq.enabled":                      "true",
						"aro.genevalogging.enabled":                "false",
						"aro.imageconfig.enabled":                  "true",
						"aro.machine.enabled":                      "false",
						"aro.machinehealthcheck.enabled":           "true",
						"aro.machinehealthcheck.managed":           "false",
						"aro.machineset.enabled":                   "true",
						"aro.monitoring.enabled":                   "false",
						"aro.nodedrainer.enabled":                  "true",
						"aro.pullsecret.enabled":                   "false",
						"aro.pullsecret.managed":                   "true",
						"aro.rbac.enabled":                         "false",
						"aro.routefix.enabled":                     "true",
						"aro.storageaccounts.enabled":              "false",
						"aro.workaround.enabled":                   "true",
						"rh.srep.muo.enabled":                      "false",
						"rh.srep.muo.managed":                      "true",
					},
				},
			},
		},
		&api.OpenShiftClusterDocument{
			ID:  fakeID_3,
			Key: fakeKey_3,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: fakeKey_3,
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: api.OperatorFlags{
						"aro.alertwebhook.enabled":                 "true",
						"aro.autosizednodes.enable":                "false",
						"aro.azuresubnets.enabled":                 "true",
						"aro.azuresubnets.nsg.managed":             "true",
						"aro.azuresubnets.serviceendpoint.managed": "true",
						"aro.banner.enabled":                       "false",
						"aro.checker.enabled":                      "true",
						"aro.dnsmasq.enabled":                      "true",
						"aro.genevalogging.enabled":                "true",
						"aro.imageconfig.enabled":                  "true",
						"aro.machine.enabled":                      "true",
						"aro.machinehealthcheck.enabled":           "true",
						"aro.machinehealthcheck.managed":           "true",
						"aro.machineset.enabled":                   "true",
						"aro.monitoring.enabled":                   "true",
						"aro.nodedrainer.enabled":                  "true",
						"aro.pullsecret.enabled":                   "true",
						"aro.pullsecret.managed":                   "true",
						"aro.rbac.enabled":                         "true",
						"aro.routefix.enabled":                     "true",
						"aro.storageaccounts.enabled":              "true",
						"aro.workaround.enabled":                   "true",
						"rh.srep.muo.enabled":                      "true",
						"rh.srep.muo.managed":                      "true",
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
