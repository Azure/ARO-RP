// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func TestAdminHiveK8sObjectsList(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		requestURL     string
		wantStatusCode int
	}

	for _, tt := range []*test{
		{
			// name param is empty → routes to listHiveK8sObjects
			// returns 500 because hiveClusterManager is nil (hive not enabled locally)
			name:           "list when name is empty",
			requestURL:     "/admin/hive/k8s/pods?namespace=default",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			// name param is set → routes to getHiveK8sObject
			// returns 500 because hiveClusterManager is nil (hive not enabled locally)
			name:           "get when name is provided",
			requestURL:     "/admin/hive/k8s/pods?namespace=default&name=testpod",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			// hiveClusterManager is nil → returns "hive is not enabled" error
			name:           "hive not enabled returns error",
			requestURL:     "/admin/hive/k8s/pods?namespace=default",
			wantStatusCode: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			// nil hiveClusterManager — hive is not enabled in local dev.
			// getHiveDynamicClient checks this first and returns
			// "hive is not enabled" error before attempting any connection.
			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("resource", "pods")
			reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			reqCtx = context.WithValue(reqCtx, middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger()))
			req = req.WithContext(reqCtx)

			f.adminHiveK8sObjectsList(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("got status %d, wanted %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}
