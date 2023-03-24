package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminKubernetesGetPodLogs(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()

	type test struct {
		name                    string
		namespace               string
		podName                 string
		containerName           string
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponse            []byte
		wantResponseContentType string
		wantError               string
	}

	for _, tt := range []*test{
		{
			name:          "cluster exist in db",
			namespace:     "openshift-image-registry",
			podName:       "config",
			containerName: "container-1",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGetPodLogs(gomock.Any(), tt.namespace, tt.podName, tt.containerName).
					Return([]byte(`fake logs`), nil)
			},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "text/plain",
			wantResponse:            []byte(`fake logs` + "\n"),
		},
		{
			name:          "AdminKubernetesGetPodLogs handles errors",
			namespace:     "openshift-image-registry",
			podName:       "config",
			containerName: "container-1",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGetPodLogs(gomock.Any(), tt.namespace, tt.podName, tt.containerName).
					Return(nil, errors.New("i/o timeout"))
			},
			wantStatusCode:          http.StatusInternalServerError,
			wantResponseContentType: "application/json",
			wantError:               "500: InternalServerError: : Internal server error.",
		},
		{
			name:                    "customer namespace, validation should fail",
			namespace:               "customer-namespace",
			podName:                 "config",
			containerName:           "container-1",
			wantStatusCode:          http.StatusForbidden,
			wantResponseContentType: "application/json",
			wantError:               "403: Forbidden: : Access to the provided namespace 'customer-namespace' is forbidden.",
		},
		{
			name:                    "empty container name",
			namespace:               "openshift-image-registry",
			podName:                 "config",
			containerName:           "",
			wantStatusCode:          http.StatusBadRequest,
			wantResponseContentType: "application/json",
			wantError:               "400: InvalidParameter: : The provided container name '' is invalid.",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", method, tt.name), func(t *testing.T) {
			resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()
			k := mock_adminactions.NewMockKubeActions(ti.controller)

			if tt.mocks != nil {
				tt.mocks(tt, k)
			}

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "0.0.0.0",
						},
					},
				},
			})
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/admin%s/kubernetespodlogs?container=%s&namespace=%s&podname=%s", resourceID, tt.containerName, tt.namespace, tt.podName),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Error(fmt.Errorf("unexpected \"Content-Type\" response header value \"%s\", wanted \"%s\"", resp.Header.Get("Content-Type"), tt.wantResponseContentType))
			}
		})
	}
}
