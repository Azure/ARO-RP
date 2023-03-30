package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-test/deep"
	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestRegionListPublic(t *testing.T) {
	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	os.Setenv("AZURE_ENVIRONMENT", azureclient.PublicCloud.Environment.Name)

	req, err := http.NewRequest("GET", "/api/regions", nil)
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

	var r RegionInfo
	err = json.NewDecoder(w.Body).Decode(&r)
	if err != nil {
		t.Fatal(err)
	}

	expected := RegionInfo{
		Regions: []Region{
			{
				Name: "australiacentral",
				URL:  "https://australiacentral.admin.aro.azure.com",
			},
			{
				Name: "australiacentral2",
				URL:  "https://australiacentral2.admin.aro.azure.com",
			},
			{
				Name: "australiaeast",
				URL:  "https://australiaeast.admin.aro.azure.com",
			},
			{
				Name: "australiasoutheast",
				URL:  "https://australiasoutheast.admin.aro.azure.com",
			},
			{
				Name: "brazilsouth",
				URL:  "https://brazilsouth.admin.aro.azure.com",
			},
			{
				Name: "brazilsoutheast",
				URL:  "https://brazilsoutheast.admin.aro.azure.com",
			},
			{
				Name: "canadacentral",
				URL:  "https://canadacentral.admin.aro.azure.com",
			},
			{
				Name: "canadaeast",
				URL:  "https://canadaeast.admin.aro.azure.com",
			},
			{
				Name: "centralindia",
				URL:  "https://centralindia.admin.aro.azure.com",
			},
			{
				Name: "centralus",
				URL:  "https://centralus.admin.aro.azure.com",
			},
			{
				Name: "centraluseuap",
				URL:  "https://centraluseuap.admin.aro.azure.com",
			},
			{
				Name: "eastasia",
				URL:  "https://eastasia.admin.aro.azure.com",
			},
			{
				Name: "eastus",
				URL:  "https://eastus.admin.aro.azure.com",
			},
			{
				Name: "eastus2",
				URL:  "https://eastus2.admin.aro.azure.com",
			},
			{
				Name: "eastus2euap",
				URL:  "https://eastus2euap.admin.aro.azure.com",
			},
			{
				Name: "francecentral",
				URL:  "https://francecentral.admin.aro.azure.com",
			},
			{
				Name: "germanywestcentral",
				URL:  "https://germanywestcentral.admin.aro.azure.com",
			},
			{
				Name: "japaneast",
				URL:  "https://japaneast.admin.aro.azure.com",
			},
			{
				Name: "japanwest",
				URL:  "https://japanwest.admin.aro.azure.com",
			},
			{
				Name: "koreacentral",
				URL:  "https://koreacentral.admin.aro.azure.com",
			},
			{
				Name: "northcentralus",
				URL:  "https://northcentralus.admin.aro.azure.com",
			},
			{
				Name: "northeurope",
				URL:  "https://northeurope.admin.aro.azure.com",
			},
			{
				Name: "norwaywest",
				URL:  "https://norwaywest.admin.aro.azure.com",
			},
			{
				Name: "norwayeast",
				URL:  "https://norwayeast.admin.aro.azure.com",
			},
			{
				Name: "qatar",
				URL:  "https://qatar.admin.aro.azure.com",
			},
			{
				Name: "southafricanorth",
				URL:  "https://southafricanorth.admin.aro.azure.com",
			},
			{
				Name: "southcentralus",
				URL:  "https://southcentralus.admin.aro.azure.com",
			},
			{
				Name: "southeastasia",
				URL:  "https://southeastasia.admin.aro.azure.com",
			},
			{
				Name: "southindia",
				URL:  "https://southindia.admin.aro.azure.com",
			},
			{
				Name: "swedencentral",
				URL:  "https://swedencentral.admin.aro.azure.com",
			},
			{
				Name: "switzerlandnorth",
				URL:  "https://switzerlandnorth.admin.aro.azure.com",
			},
			{
				Name: "switzerlandwest",
				URL:  "https://switzerlandwest.admin.aro.azure.com",
			},
			{
				Name: "uaenorth",
				URL:  "https://uaenorth.admin.aro.azure.com",
			},
			{
				Name: "uaecentral",
				URL:  "https://uaecentral.admin.aro.azure.com",
			},
			{
				Name: "uksouth",
				URL:  "https://uksouth.admin.aro.azure.com",
			},
			{
				Name: "ukwest",
				URL:  "https://ukwest.admin.aro.azure.com",
			},
			{
				Name: "westcentralus",
				URL:  "https://westcentralus.admin.aro.azure.com",
			},
			{
				Name: "westeurope",
				URL:  "https://westeurope.admin.aro.azure.com",
			},
			{
				Name: "westus",
				URL:  "https://westus.admin.aro.azure.com",
			},
			{
				Name: "westus2",
				URL:  "https://westus2.admin.aro.azure.com",
			},
			{
				Name: "westus3",
				URL:  "https://westus3.admin.aro.azure.com",
			},
		},
	}

	for _, l := range deep.Equal(expected, r) {
		t.Error(l)
	}
}

func TestRegionListFF(t *testing.T) {
	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixture := testdatabase.NewFixture().
		WithOpenShiftClusters(dbOpenShiftClusters)

	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	p := &portal{
		dbOpenShiftClusters: dbOpenShiftClusters,
	}

	os.Setenv("AZURE_ENVIRONMENT", azureclient.USGovernmentCloud.Environment.Name)

	req, err := http.NewRequest("GET", "/api/regions", nil)
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

	var r RegionInfo
	err = json.NewDecoder(w.Body).Decode(&r)
	if err != nil {
		t.Fatal(err)
	}

	expected := RegionInfo{
		Regions: []Region{},
	}

	for _, l := range deep.Equal(expected, r) {
		t.Error(l)
	}
}
