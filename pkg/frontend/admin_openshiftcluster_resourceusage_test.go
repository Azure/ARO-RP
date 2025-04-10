package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_frontend "github.com/Azure/ARO-RP/pkg/util/mocks/frontend"
)

func TestGetAdminTopPods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKA := mock_adminactions.NewMockKubeActions(ctrl)
	mockDB := mock_frontend.NewMockOpenShiftClusters(ctrl)
	mockDBGroup := mock_frontend.NewMockfrontendDBs(ctrl)

	mockClusterID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster"
	doc := &api.OpenShiftClusterDocument{
		Key: mockClusterID,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: mockClusterID,
			Properties: api.OpenShiftClusterProperties{
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "1.2.3.4",
				},
				AdminKubeconfig: []byte(`
apiVersion: v1
clusters:
- cluster:
    server: https://1.2.3.4:443
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
kind: Config
preferences: {}
users:
- name: test
  user:
    token: dummy
`),
			},
		},
	}

	mockDBGroup.EXPECT().OpenShiftClusters().Return(mockDB, nil)
	mockDB.EXPECT().Get(gomock.Any(), mockClusterID).Return(doc, nil)
	mockKA.EXPECT().TopPods(gomock.Any(), gomock.Any(), true).Return([]adminactions.PodMetrics{
		{
			Namespace:        "default",
			PodName:          "pod-1",
			NodeName:         "node-1",
			CPUUsage:         "100m",
			MemoryUsage:      "200Mi",
			CPUPercentage:    10.0,
			MemoryPercentage: 20.0,
		},
	}, nil)

	f := &frontend{
		dbGroup: mockDBGroup,
		kubeActionsFactory: func(log *logrus.Entry, _ env.Interface, _ *api.OpenShiftCluster) (adminactions.KubeActions, error) {
			return mockKA, nil
		},
		env: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/admin"+mockClusterID+"/top/pods", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger())))
	rr := httptest.NewRecorder()

	f.getAdminTopPods(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}
}

func TestGetAdminTopNodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKA := mock_adminactions.NewMockKubeActions(ctrl)
	mockDB := mock_frontend.NewMockOpenShiftClusters(ctrl)
	mockDBGroup := mock_frontend.NewMockfrontendDBs(ctrl)

	mockClusterID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster"
	doc := &api.OpenShiftClusterDocument{
		Key: mockClusterID,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: mockClusterID,
			Properties: api.OpenShiftClusterProperties{
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "1.2.3.4",
				},
				AdminKubeconfig: []byte(`
apiVersion: v1
clusters:
- cluster:
    server: https://1.2.3.4:443
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
kind: Config
preferences: {}
users:
- name: test
  user:
    token: dummy
`),
			},
		},
	}

	mockDBGroup.EXPECT().OpenShiftClusters().Return(mockDB, nil)
	mockDB.EXPECT().Get(gomock.Any(), mockClusterID).Return(doc, nil)
	mockKA.EXPECT().TopNodes(gomock.Any(), gomock.Any()).Return([]adminactions.NodeMetrics{
		{
			NodeName:         "node-1",
			CPUUsage:         "500m",
			MemoryUsage:      "1Gi",
			CPUPercentage:    25,
			MemoryPercentage: 50,
		},
	}, nil)

	f := &frontend{
		dbGroup: mockDBGroup,
		kubeActionsFactory: func(log *logrus.Entry, _ env.Interface, _ *api.OpenShiftCluster) (adminactions.KubeActions, error) {
			return mockKA, nil
		},
		env: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/admin"+mockClusterID+"/top/nodes", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger())))
	rr := httptest.NewRecorder()

	f.getAdminTopNodes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}
}
