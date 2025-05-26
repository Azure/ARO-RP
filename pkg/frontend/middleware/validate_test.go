package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/test/validate"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
)

func emptyResponse(w http.ResponseWriter, r *http.Request) {}

func TestValidate(t *testing.T) {
	ValidateMiddleware := ValidateMiddleware{
		Location: "",
		Apis:     api.APIs,
	}

	chiRouter := chi.NewMux()
	chiRouter.Route("/subscriptions/{subscriptionId}", func(r chi.Router) {
		r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}", func(r chi.Router) {
			r.Use(ValidateMiddleware.Validate)
			r.Put("/", emptyResponse)
		})

		r.Route("/", func(r chi.Router) {
			r.Use(ValidateMiddleware.Validate)
			r.Put("/", emptyResponse)
		})
	})

	chiRouter.Route("/providers/{resourceProviderNamespace}/operations", func(r chi.Router) {
		r.Use(ValidateMiddleware.Validate)
		r.Put("/", emptyResponse)
	})

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
			name: "valid operations",
			path: "/providers/microsoft.redhatopenshift/operations?api-version=2020-04-30",
		},
		{
			name:    "invalid operations - resourceProviderNamespace",
			path:    "/providers/invalid/operations?api-version=2020-04-30",
			wantErr: "400: InvalidResourceNamespace: : The resource namespace 'invalid' is invalid.",
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

			chiRouter.ServeHTTP(w, r)

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
					t.Fatalf("got %s while unmarshalling. status code : %d", err, w.Code)
				}
				cloudErr.StatusCode = w.Code

				validate.CloudError(t, cloudErr)

				if tt.wantErr != cloudErr.Error() {
					t.Errorf("wanted %s but got %s", tt.wantErr, cloudErr.Error())
				}
			}
		})
	}
}
