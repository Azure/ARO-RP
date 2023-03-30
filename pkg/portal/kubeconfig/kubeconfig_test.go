package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestNew(t *testing.T) {
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster"
	elevatedGroupIDs := []string{"10000000-0000-0000-0000-000000000000"}
	username := "username"
	password := "03030303-0303-0303-0303-030303030001"

	servingCert := &x509.Certificate{}

	for _, tt := range []struct {
		name           string
		r              func(*http.Request)
		elevated       bool
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakePortalDocumentClient)
		wantStatusCode int
		wantHeaders    http.Header
		wantBody       string
	}{
		{
			name: "success - not elevated",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  password,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				checker.AddPortalDocuments(portalDocument)
			},
			wantStatusCode: http.StatusOK,
			wantHeaders: http.Header{
				"Content-Disposition": []string{`attachment; filename="cluster.kubeconfig"`},
			},
			wantBody: "{\n    \"kind\": \"Config\",\n    \"apiVersion\": \"v1\",\n    \"preferences\": {},\n    \"clusters\": [\n        {\n            \"name\": \"cluster\",\n            \"cluster\": {\n                \"server\": \"https://localhost:8444/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/kubeconfig/proxy\",\n                \"certificate-authority-data\": \"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\"\n            }\n        }\n    ],\n    \"users\": [\n        {\n            \"name\": \"user\",\n            \"user\": {\n                \"token\": \"03030303-0303-0303-0303-030303030001\"\n            }\n        }\n    ],\n    \"contexts\": [\n        {\n            \"name\": \"context\",\n            \"context\": {\n                \"cluster\": \"cluster\",\n                \"user\": \"user\",\n                \"namespace\": \"default\"\n            }\n        }\n    ],\n    \"current-context\": \"context\"\n}",
		},
		{
			name:     "success - elevated",
			elevated: true,
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  password,
					TTL: 21600,
					Portal: &api.Portal{
						Username: username,
						ID:       resourceID,
						Kubeconfig: &api.Kubeconfig{
							Elevated: true,
						},
					},
				}
				checker.AddPortalDocuments(portalDocument)
			},
			wantStatusCode: http.StatusOK,
			wantHeaders: http.Header{
				"Content-Disposition": []string{`attachment; filename="cluster-elevated.kubeconfig"`},
			},
			wantBody: "{\n    \"kind\": \"Config\",\n    \"apiVersion\": \"v1\",\n    \"preferences\": {},\n    \"clusters\": [\n        {\n            \"name\": \"cluster\",\n            \"cluster\": {\n                \"server\": \"https://localhost:8444/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/kubeconfig/proxy\",\n                \"certificate-authority-data\": \"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\"\n            }\n        }\n    ],\n    \"users\": [\n        {\n            \"name\": \"user\",\n            \"user\": {\n                \"token\": \"03030303-0303-0303-0303-030303030001\"\n            }\n        }\n    ],\n    \"contexts\": [\n        {\n            \"name\": \"context\",\n            \"context\": {\n                \"cluster\": \"cluster\",\n                \"user\": \"user\",\n                \"namespace\": \"default\"\n            }\n        }\n    ],\n    \"current-context\": \"context\"\n}",
		},
		{
			name: "bad path",
			r: func(r *http.Request) {
				r.URL.Path = "/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/kubeconfig/new"
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid resourceId \"/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster\"\n",
		},
		{
			name: "sad database",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalClient.SetError(fmt.Errorf("sad"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "Internal Server Error\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			dbPortal, portalClient := testdatabase.NewFakePortal()

			fixture := testdatabase.NewFixture().
				WithPortal(dbPortal)

			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, portalClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			ctx = context.WithValue(ctx, middleware.ContextKeyUsername, username)
			if tt.elevated {
				ctx = context.WithValue(ctx, middleware.ContextKeyGroups, elevatedGroupIDs)
			} else {
				ctx = context.WithValue(ctx, middleware.ContextKeyGroups, []string(nil))
			}
			r, err := http.NewRequestWithContext(ctx, http.MethodPost,
				"https://localhost:8444"+resourceID+"/kubeconfig/new", nil)
			if err != nil {
				panic(err)
			}

			r.Header.Set("Content-Type", "application/json")

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_env := mock_env.NewMockInterface(ctrl)
			_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			_env.EXPECT().Hostname().AnyTimes().Return("testhost")
			_env.EXPECT().Location().AnyTimes().Return("eastus")

			aadAuthenticatedRouter := &mux.Router{}

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			k := New(baseLog, audit, _env, baseAccessLog, servingCert, elevatedGroupIDs, nil, dbPortal, nil)

			if tt.r != nil {
				tt.r(r)
			}

			w := responsewriter.New(r)

			aadAuthenticatedRouter.NewRoute().Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/new").HandlerFunc(k.New)

			aadAuthenticatedRouter.ServeHTTP(w, r)

			portalClient.SetError(nil)

			for _, err = range checker.CheckPortals(portalClient) {
				t.Error(err)
			}

			resp := w.Response()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			for k, v := range tt.wantHeaders {
				if !reflect.DeepEqual(resp.Header[k], v) {
					t.Errorf(k, resp.Header[k], v)
				}
			}

			wantContentType := "application/json"
			if resp.StatusCode != http.StatusOK {
				wantContentType = "text/plain; charset=utf-8"
			}
			if resp.Header.Get("Content-Type") != wantContentType {
				t.Error(resp.Header.Get("Content-Type"))
			}

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(b) != tt.wantBody {
				t.Errorf("%q", string(b))
			}
		})
	}
}
