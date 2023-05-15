package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestNew(t *testing.T) {
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster"
	elevatedGroupIDs := []string{"10000000-0000-0000-0000-000000000000"}
	username := "username"
	password := "03030303-0303-0303-0303-030303030001"
	master := 0

	hostKey, _, err := utiltls.GenerateKeyAndCertificate("proxy", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name           string
		r              func(*http.Request)
		checker        func(*testdatabase.Checker, *cosmosdb.FakePortalDocumentClient)
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "success",
			checker: func(checker *testdatabase.Checker, portalClient *cosmosdb.FakePortalDocumentClient) {
				checker.AddPortalDocuments(&api.PortalDocument{
					ID:  password,
					TTL: 60,
					Portal: &api.Portal{
						Username: username,
						ID:       resourceID,
						SSH: &api.SSH{
							Master: master,
						},
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "{\n    \"command\": \"ssh username@localhost\",\n    \"password\": \"03030303-0303-0303-0303-030303030001\"\n}",
		},
		{
			name: "bad path",
			r: func(r *http.Request) {
				r.URL.Path = "/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/ssh/new"
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid resourceId \"/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster\"\n",
		},
		{
			name: "bad content type",
			r: func(r *http.Request) {
				r.Header.Set("Content-Type", "bad")
			},
			wantStatusCode: http.StatusUnsupportedMediaType,
			wantBody:       "Unsupported Media Type\n",
		},
		{
			name: "empty request",
			r: func(r *http.Request) {
				r.Body = io.NopCloser(bytes.NewReader(nil))
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Bad Request\n",
		},
		{
			name: "junk request",
			r: func(r *http.Request) {
				r.Body = io.NopCloser(strings.NewReader("{{"))
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Bad Request\n",
		},
		{
			name: "not elevated",
			r: func(r *http.Request) {
				*r = *r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyGroups, []string{}))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "{\n    \"error\": \"Elevated access is required.\"\n}",
		},
		{
			name: "sad database",
			checker: func(checker *testdatabase.Checker, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalClient.SetError(fmt.Errorf("sad"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "Internal Server Error\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			dbPortal, portalClient := testdatabase.NewFakePortal()

			checker := testdatabase.NewChecker()

			if tt.checker != nil {
				tt.checker(checker, portalClient)
			}

			ctx = context.WithValue(ctx, middleware.ContextKeyUsername, username)
			ctx = context.WithValue(ctx, middleware.ContextKeyGroups, elevatedGroupIDs)
			r, err := http.NewRequestWithContext(ctx, http.MethodPost,
				"https://localhost:8444"+resourceID+"/ssh/new", strings.NewReader(fmt.Sprintf(`{"master":%d}`, master)))
			if err != nil {
				panic(err)
			}

			r.Header.Set("Content-Type", "application/json")

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			env := mock_env.NewMockCore(ctrl)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			s, err := New(env, logrus.NewEntry(logrus.StandardLogger()), nil, nil, hostKey, elevatedGroupIDs, nil, dbPortal, nil)
			if err != nil {
				t.Fatal(err)
			}

			router := mux.NewRouter()
			router.Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/ssh/new").HandlerFunc(s.New)

			if tt.r != nil {
				tt.r(r)
			}

			w := responsewriter.New(r)

			router.ServeHTTP(w, r)

			portalClient.SetError(nil)

			for _, err = range checker.CheckPortals(portalClient) {
				t.Error(err)
			}

			resp := w.Response()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
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
