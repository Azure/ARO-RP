package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

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

	// Inject chi params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("region", "eastus")
	rctx.URLParams.Add("resource", "pods")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Inject logger
	log := logrus.NewEntry(logrus.StandardLogger())
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, log))

	rr := httptest.NewRecorder()

	// Call handler directly
	f.adminHiveK8sObjectsList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
