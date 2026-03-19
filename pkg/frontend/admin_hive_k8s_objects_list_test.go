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
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func TestAdminHiveK8sObjectsList(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		requestURL     string
		hiveEnabled    bool
		mocks          func(mockHive *mock_hive.MockClusterManager)
		wantStatusCode int
	}

	for _, tt := range []*test{
		{
			// name param is empty → must call ListK8sObjects
			name:        "list when name is empty",
			requestURL:  "/admin/hive/k8s/pods?namespace=default",
			hiveEnabled: true,
			mocks: func(mockHive *mock_hive.MockClusterManager) {
				mockHive.EXPECT().
					ListHiveK8sObjects(gomock.Any(), "pods", "default").
					Return([]byte(`{}`), nil).
					Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			// name param is set → must call GetK8sObject
			name:        "get when name is provided",
			requestURL:  "/admin/hive/k8s/pods?namespace=default&name=testpod",
			hiveEnabled: true,
			mocks: func(mockHive *mock_hive.MockClusterManager) {
				mockHive.EXPECT().
					GetHiveK8sObject(gomock.Any(), "pods", "default", "testpod").
					Return([]byte(`{}`), nil).
					Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			// hiveClusterManager is nil → returns error
			name:           "hive not enabled returns error",
			requestURL:     "/admin/hive/k8s/pods?namespace=default",
			hiveEnabled:    false,
			mocks:          func(mockHive *mock_hive.MockClusterManager) {},
			wantStatusCode: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			controller := gomock.NewController(t)
			defer ti.done()
			defer controller.Finish()

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			var f *frontend

			if tt.hiveEnabled {
				clusterManager := mock_hive.NewMockClusterManager(controller)
				tt.mocks(clusterManager)
				f, err = NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, clusterManager, nil, nil, nil, nil, nil)
			} else {
				f, err = NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			}
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
