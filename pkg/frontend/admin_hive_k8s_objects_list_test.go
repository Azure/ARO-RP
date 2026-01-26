package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	mock_hive "github.com/Azure/ARO-RP/pkg/frontend/mocks"
)

func TestAdminHiveK8sObjectsList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock manager
	manager := mock_hive.NewMockHiveK8sObjectManager(ctrl)
	manager.EXPECT().
		List(gomock.Any(), "eastus", "pods").
		Return([]byte(`{"items":[]}`), nil)

	// Minimal frontend (NO NewFrontend)
	f := &frontend{
		hiveK8sObjectManager: manager,
	}

	// Request
	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/eastus/pods",
		nil,
	)

	// Inject chi route params
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("region", "eastus")
	routeCtx.URLParams.Add("resource", "pods")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	// Inject logger (REQUIRED)
	log := logrus.New().WithField("test", "list")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, log))

	rr := httptest.NewRecorder()

	// Call handler
	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminHiveK8sObjectsGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock manager
	manager := mock_hive.NewMockHiveK8sObjectManager(ctrl)
	manager.EXPECT().
		Get(gomock.Any(), "eastus", "pods", "mypod").
		Return([]byte(`{"metadata":{"name":"mypod"}}`), nil)

	// Minimal frontend
	f := &frontend{
		hiveK8sObjectManager: manager,
	}

	// Request
	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/eastus/pods?name=mypod",
		nil,
	)

	// Inject chi route params
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("region", "eastus")
	routeCtx.URLParams.Add("resource", "pods")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	// Inject logger (REQUIRED)
	log := logrus.New().WithField("test", "get")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, log))

	rr := httptest.NewRecorder()

	// Call handler
	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
