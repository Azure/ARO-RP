package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/test/validate"
)

func TestValidate(t *testing.T) {
	router := mux.NewRouter()

	router.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	router.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	router.
		Path("/subscriptions/{subscriptionId}").
		Queries("api-version", "2.0").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	ValidateMiddleware := ValidateMiddleware{
		Location: "",
		Apis:     api.APIs,
	}
	router.Use(ValidateMiddleware.Validate)

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name: "valid openShiftClusters",
			path: "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster?api-version=2020-04-30",
		},
		{
			name:    "invalid openShiftClusters - case",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/Resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster?api-version=2020-04-30",
			wantErr: "500: InternalServerError: : Internal server error.",
		},
		{
			name:    "invalid openShiftClusters - subscriptionId",
			path:    "/subscriptions/invalid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster?api-version=2020-04-30",
			wantErr: "400: InvalidSubscriptionID: : The provided subscription identifier 'invalid' is malformed or invalid.",
		},
		{
			name:    "invalid openShiftClusters - resourceGroupName",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/$/providers/microsoft.redhatopenshift/openshiftclusters/cluster?api-version=2020-04-30",
			wantErr: "400: ResourceGroupNotFound: : Resource group '$' could not be found.",
		},
		{
			name:    "invalid openShiftClusters - resourceProviderNamespace",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourcegroup/providers/invalid/openshiftclusters/cluster?api-version=2020-04-30",
			wantErr: "400: InvalidResourceNamespace: : The resource namespace 'invalid' is invalid.",
		},
		{
			name:    "invalid openShiftClusters - resourceType",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/invalid/cluster?api-version=2020-04-30",
			wantErr: "400: InvalidResourceType: : The resource type 'invalid' could not be found in the namespace 'microsoft.redhatopenshift' for api version '2020-04-30'.",
		},
		{
			name:    "invalid openShiftClusters - resourceName",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/$?api-version=2020-04-30",
			wantErr: "400: ResourceNotFound: : The Resource 'microsoft.redhatopenshift/openshiftclusters/$' under resource group 'resourcegroup' was not found.",
		},
		{
			name:    "invalid openShiftClusters - api-version",
			path:    "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster?api-version=invalid",
			wantErr: "400: InvalidResourceType: : The resource type 'openshiftclusters' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
		{
			name: "valid operations",
			path: "/providers/microsoft.redhatopenshift/operations?api-version=2020-04-30",
		},
		{
			name:    "invalid operations - resourceProviderNamespace",
			path:    "/providers/invalid/operations?api-version=2020-04-30",
			wantErr: "400: InvalidResourceNamespace: : The resource namespace 'invalid' is invalid.",
		},
		{
			name:    "invalid operations - api-version",
			path:    "/providers/microsoft.redhatopenshift/operations?api-version=invalid",
			wantErr: "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
		{
			name: "valid subscriptions",
			path: "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491?api-version=2.0",
		},
		{
			name:    "invalid subscriptions - subscriptionId",
			path:    "/subscriptions/invalid?api-version=2.0",
			wantErr: "400: InvalidSubscriptionID: : The provided subscription identifier 'invalid' is malformed or invalid.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			if tt.wantErr == "" {
				if w.Code != http.StatusOK {
					t.Error(w.Code)
				}

				if w.Body.String() != "" {
					t.Error(w.Body.String())
				}
			} else {
				t.Log(w.Body.String())
				var cloudErr *api.CloudError
				err := json.Unmarshal(w.Body.Bytes(), &cloudErr)
				if err != nil {
					t.Fatal(err)
				}
				cloudErr.StatusCode = w.Code

				validate.CloudError(t, cloudErr)

				if tt.wantErr != cloudErr.Error() {
					t.Error(cloudErr)
				}
			}
		})
	}
}
