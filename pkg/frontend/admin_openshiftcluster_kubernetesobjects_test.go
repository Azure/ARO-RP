package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminKubernetesObjectsGetAndDelete(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		objKind        string
		objNamespace   string
		objName        string
		force          string
		mocks          func(*test, *mock_adminactions.MockKubeActions)
		method         string
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			method:       http.MethodGet,
			name:         "cluster exist in db - get",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), tt.objKind, tt.objNamespace, tt.objName).
					Return([]byte(`{"Kind": "test"}`), nil)
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "configmaps"}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"Kind": "test"}` + "\n"),
		},
		{
			method:       http.MethodGet,
			name:         "cluster exist in db - list",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), tt.objKind, tt.objNamespace).
					Return([]byte(`{"Kind": "test"}`), nil)
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "configmaps"}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"Kind": "test"}` + "\n"),
		},
		{
			method:       http.MethodGet,
			name:         "no groupKind provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objKind)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided resource is invalid.",
		},
		{
			method:       http.MethodGet,
			name:         "secret requested",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Secret",
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "secrets"}, nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			method:       http.MethodDelete,
			name:         "cluster exist in db",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeDelete(gomock.Any(), tt.objKind, tt.objNamespace, tt.objName, false).
					Return(nil)
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "configmaps"}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			method:       http.MethodDelete,
			name:         "force delete pod",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Pod",
			objNamespace: "openshift",
			objName:      "aro-pod",
			force:        "true",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeDelete(gomock.Any(), tt.objKind, tt.objNamespace, tt.objName, true).
					Return(nil)
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "pods"}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			method:       http.MethodDelete,
			name:         "force delete not allowed",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift",
			objName:      "config",
			force:        "true",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Force deleting groupKind 'ConfigMap' is forbidden.",
		},
		{
			method:       http.MethodDelete,
			name:         "no groupKind provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objKind)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided resource is invalid.",
		},
		{
			method:       http.MethodDelete,
			name:         "no name provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "this",
			objNamespace: "openshift",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objKind)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided resource is invalid.",
		},
		{
			method:       http.MethodDelete,
			name:         "secret requested",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Secret",
			objNamespace: "openshift",
			objName:      "config",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objKind).Return(&schema.GroupVersionResource{Resource: "secrets"}, nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", tt.method, tt.name), func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, k)

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			requestStr := fmt.Sprintf("https://server/admin%s/kubernetesObjects?kind=%s&namespace=%s&name=%s", tt.resourceID, tt.objKind, tt.objNamespace, tt.objName)
			if tt.method == http.MethodDelete && tt.force != "" {
				requestStr = fmt.Sprintf("%s&force=%s", requestStr, tt.force)
			}

			resp, b, err := ti.request(tt.method,
				requestStr,
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAdminPostKubernetesObjects(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode int
		objInBody      *unstructured.Unstructured
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: resourceID,
			objInBody: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), tt.objInBody).
					Return(nil)
				k.EXPECT().ResolveGVR(tt.objInBody.GetKind()).Return(&schema.GroupVersionResource{Resource: "configmaps"}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:       "secret requested",
			resourceID: resourceID,
			objInBody: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Secret",
					"metadata": map[string]interface{}{
						"namespace": "openshift",
						"name":      "secret",
					},
				},
			},
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().ResolveGVR(tt.objInBody.GetKind()).Return(&schema.GroupVersionResource{Resource: "secrets"}, nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, k)

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/kubernetesObjects", tt.resourceID),
				http.Header{
					"Content-Type": []string{"application/json"},
				}, tt.objInBody)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
