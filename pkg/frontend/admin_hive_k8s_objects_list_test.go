package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func withLog(req *http.Request) *http.Request {
	log := logrus.New().WithField("test", "hive-k8sobjects")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyLog, log)
	return req.WithContext(ctx)
}

func withChiRouteParam(req *http.Request, key, val string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestAdminHiveK8sObjectsList_ManagerNil_DefaultBehavior(t *testing.T) {

	t.Setenv(envLocalDevMockHive, "")

	f := &frontend{
		hiveK8sObjectManager: nil,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/pods?namespace=default",
		nil,
	)

	req = withChiRouteParam(req, "resource", "pods")
	req = withLog(req)

	rr := httptest.NewRecorder()

	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusNotImplemented, rr.Code)
	require.Contains(t, rr.Body.String(), "hive k8s object manager not configured")
}

func TestAdminHiveK8sObjectsList_ManagerNil_MockEnabled_ReturnsMock(t *testing.T) {

	t.Setenv(envLocalDevMockHive, "true")

	f := &frontend{
		hiveK8sObjectManager: nil,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/hive/k8sobjects/pods?namespace=default",
		nil,
	)

	req = withChiRouteParam(req, "resource", "pods")
	req = withLog(req)

	rr := httptest.NewRecorder()

	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "local-dev-pod")
}
