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

	manager := mock_hive.NewMockHiveK8sObjectManager(ctrl)
	manager.EXPECT().
		List(gomock.Any(), "pods", "default").
		Return([]byte(`{"items":[]}`), nil)

	f := &frontend{
		hiveK8sObjectManager: manager,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/pods?namespace=default",
		nil,
	)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("resource", "pods")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	log := logrus.New().WithField("test", "list")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, log))

	rr := httptest.NewRecorder()
	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminHiveK8sObjectsGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	manager := mock_hive.NewMockHiveK8sObjectManager(ctrl)
	manager.EXPECT().
		Get(gomock.Any(), "pods", "default", "mypod").
		Return([]byte(`{"metadata":{"name":"mypod"}}`), nil)

	f := &frontend{
		hiveK8sObjectManager: manager,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/pods?namespace=default&name=mypod",
		nil,
	)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("resource", "pods")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	log := logrus.New().WithField("test", "get")
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, log))

	rr := httptest.NewRecorder()
	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
