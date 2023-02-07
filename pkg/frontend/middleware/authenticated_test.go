package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestAuthenticatedForOCMAPIs(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	_env := mock_env.NewMockInterface(controller)
	r := mux.NewRouter()
	// mimic what the setupRouter func will do for this specific path
	r.HandleFunc("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}",
		func(w http.ResponseWriter, request *http.Request) {})

	r.HandleFunc("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/{ocmResourceType}/{ocmResourceName}",
		func(w http.ResponseWriter, request *http.Request) {}).
		Queries("api-version", "")
	basePath := "https://server/subscriptions/0000-0000/resourcegroups/testrg/providers/testrpn/testrt/testrn/%s/myResource?api-version=2022-09-04"

	for _, tt := range []struct {
		name                 string
		method               string
		ocmResourceType      string
		systemDataHeader     string
		wantStatus           int
		isValid              bool
		expectedValidateCall bool
	}{
		{
			name:                 "non ocm api called, system data header is not validated",
			method:               "GET",
			ocmResourceType:      "",
			systemDataHeader:     `{"systemData":{"lastModifiedBy":"unused"}}`,
			wantStatus:           200,
			isValid:              true,
			expectedValidateCall: false,
		},
		{
			name:                 "ocm api 'syncsets' validator returns true, success",
			method:               "GET",
			ocmResourceType:      "syncsets",
			systemDataHeader:     `{"systemData":{"lastModifiedBy":"abc-123"}}`,
			wantStatus:           200,
			isValid:              true,
			expectedValidateCall: true,
		},
		{
			name:                 "ocm api 'syncsets' validator returns false, forbidden",
			method:               "GET",
			ocmResourceType:      "syncsets",
			systemDataHeader:     `{"systemData":{"lastModifiedBy":"abc-123"}}`,
			wantStatus:           403,
			isValid:              false,
			expectedValidateCall: true,
		},
	} {
		var req *http.Request
		var err error
		vars := map[string]string{
			"api-version": "2022-09-04",
		}
		if tt.ocmResourceType == "" {
			// non ocm api
			req, err = http.NewRequest("GET", "https://server/subscriptions/0000-0000/resourcegroups/testrg/providers/testrpn/testrt/testrn", nil)
		} else {
			req, err = http.NewRequest("GET", fmt.Sprintf(basePath, tt.ocmResourceType), nil)
			vars["ocmResourceType"] = tt.ocmResourceType
		}
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(ArmSystemDataHeaderKey, tt.systemDataHeader)

		req = mux.SetURLVars(req, vars)

		_env.EXPECT().ArmClientAuthorizer().Return(clientauthorizer.NewAll())
		if tt.expectedValidateCall {
			_env.EXPECT().ValidateOCMClientID(req.Header.Get(ArmSystemDataHeaderKey)).Return(tt.isValid)
		} else {
			_env.EXPECT().ValidateOCMClientID(req.Header.Get(ArmSystemDataHeaderKey)).Times(0)
		}

		rr := httptest.NewRecorder()

		Authenticated(_env)(r).ServeHTTP(rr, req)

		if status := rr.Code; status != tt.wantStatus {
			t.Fatalf("expected status: %d got: %d", tt.wantStatus, status)
		}
	}
}
