package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
)

func TestAuthenticatedForOCMAPIs(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	validOCMClientIds := []string{"abc-123"}
	chiRouter := chi.NewMux()
	chiRouter.Route("/subscriptions/{subscriptionId}", func(r chi.Router) {
		r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}", func(r chi.Router) {
			r.Use(OCMValidator{ValidOCMClientIDs: []string{}}.ValidateOCMClient)
			r.Get("/", emptyResponse)
		})
	})
	chiRouter.Route("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/{ocmResourceType}/{ocmResourceName}", func(r chi.Router) {
		r.Use(OCMValidator{ValidOCMClientIDs: validOCMClientIds}.ValidateOCMClient)
		r.Get("/", emptyResponse)
	})

	basePath := "https://server/subscriptions/0000-0000/resourcegroups/testrg/providers/testrpn/testrt/testrn/%s/myResource?api-version=2022-09-04"

	for _, tt := range []struct {
		name             string
		method           string
		ocmResourceType  string
		systemDataHeader string
		wantStatus       int
	}{
		{
			name:             "non ocm api called, system data header is not validated",
			method:           "GET",
			systemDataHeader: `{"lastModifiedBy":"unused"}`,
			wantStatus:       200,
		},
		{
			name:             "ocm api 'syncsets' validator returns true, success",
			method:           "GET",
			ocmResourceType:  "syncsets",
			systemDataHeader: `{"lastModifiedBy":"abc-123", "lastModifiedByType": "Application"}`,
			wantStatus:       200,
		},
		{
			name:             "missing lastModifiedByType, forbidden",
			method:           "GET",
			ocmResourceType:  "syncsets",
			systemDataHeader: `{"lastModifiedBy":"abc-123"}`,
			wantStatus:       403,
		},
		{
			name:             "incorrect clientID, forbidden",
			method:           "GET",
			ocmResourceType:  "syncsets",
			systemDataHeader: `{"lastModifiedBy":"def-345", "lastModifiedByType": "Application"}`,
			wantStatus:       403,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var r *http.Request
			var err error

			if tt.ocmResourceType == "" {
				// non ocm api
				r = httptest.NewRequest(http.MethodGet, "https://server/subscriptions/0000-0000/resourcegroups/testrg/providers/testrpn/testrt/testrn", nil)
			} else {
				r = httptest.NewRequest(http.MethodGet, fmt.Sprintf(basePath, tt.ocmResourceType), nil)

			}
			if err != nil {
				t.Fatal(err)
			}

			r.Header.Set(ArmSystemDataHeaderKey, tt.systemDataHeader)

			w := httptest.NewRecorder()

			chiRouter.ServeHTTP(w, r)
			if status := w.Code; status != tt.wantStatus {
				t.Fatalf("expected status: %d got: %d", tt.wantStatus, status)
			}
		})
	}
}
