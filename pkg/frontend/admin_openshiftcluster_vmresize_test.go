package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminVMResize(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		vmName         string
		vmSize         string
		fixture        func(f *testdatabase.Fixture)
		mocks          func(*test, *mock_adminactions.MockAzureActions, *mock_adminactions.MockKubeActions)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			vmName:     "aro-fake-node-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions, k *mock_adminactions.MockKubeActions) {
				node := corev1.Node{
					TypeMeta: metav1.TypeMeta{
						Kind: "node",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-fake-node-0",

						Labels: map[string]string{
							"node-role.kubernetes.io/master": "true",
						},
					},
				}
				nodeList := corev1.NodeList{
					TypeMeta: metav1.TypeMeta{
						Kind: "List",
					},
					Items: []corev1.Node{node},
				}
				marsh, _ := json.Marshal(nodeList)
				k.EXPECT().KubeList(gomock.Any(), "node", "").Return(marsh, nil)
				a.EXPECT().VMResize(gomock.Any(), tt.vmName, tt.vmSize).Return(nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:       "cluster not found",
			vmName:     "aro-fake-node-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "subscription doc not found",
			vmName:     "aro-fake-node-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})
			},
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf(`400: InvalidSubscriptionState: : Request is not allowed in unregistered subscription '%s'.`, mockSubID),
		},
		{
			name:       "master node not found",
			vmName:     "aro-fake-node-0",
			vmSize:     "Standard_D8s_v3",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions, k *mock_adminactions.MockKubeActions) {
				node := corev1.Node{
					TypeMeta: metav1.TypeMeta{
						Kind: "node",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-fake-node-0",

						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "true",
						},
					},
				}
				nodeList := corev1.NodeList{
					TypeMeta: metav1.TypeMeta{
						Kind: "List",
					},
					Items: []corev1.Node{node},
				}
				marsh, _ := json.Marshal(nodeList)
				k.EXPECT().KubeList(gomock.Any(), "node", "").Return(marsh, nil)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: : "The master node 'aro-fake-node-0' under resource group 'resourcegroup' was not found."`,
		},
		{
			name:           "invalid vmname",
			vmName:         "%26&ampersandvmname",
			resourceID:     testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:         "Standard_D8s_v3",
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions, k *mock_adminactions.MockKubeActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : The provided vmName '&' is invalid.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, a, k)

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil,
				func(e *logrus.Entry, i env.Interface, osc *api.OpenShiftCluster) (adminactions.KubeActions, error) {
					return k, nil
				}, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
					return a, nil
				}, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/resize?vmName=%s&vmSize=%s", tt.resourceID, tt.vmName, tt.vmSize),
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
