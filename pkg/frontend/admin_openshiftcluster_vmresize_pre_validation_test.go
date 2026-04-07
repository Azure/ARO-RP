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

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func fakeClusterOperatorJSON(name string, conditions []configv1.ClusterOperatorStatusCondition) []byte {
	co := configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: configv1.ClusterOperatorStatus{
			Conditions: conditions,
		},
	}
	b, _ := json.Marshal(co)
	return b
}

func healthyKubeAPIServerJSON() []byte {
	return fakeClusterOperatorJSON("kube-apiserver", []configv1.ClusterOperatorStatusCondition{
		{Type: configv1.OperatorAvailable, Status: configv1.ConditionTrue},
		{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse},
		{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse},
	})
}

func fakeAROClusterJSON(conditions []operatorv1.OperatorCondition) []byte {
	cluster := arov1alpha1.Cluster{
		Status: arov1alpha1.ClusterStatus{
			Conditions: conditions,
		},
	}
	b, _ := json.Marshal(cluster)
	return b
}

func validServicePrincipalJSON() []byte {
	return fakeAROClusterJSON([]operatorv1.OperatorCondition{
		{Type: arov1alpha1.ServicePrincipalValid, Status: operatorv1.ConditionTrue},
	})
}

func healthyEtcdJSON() []byte {
	return fakeClusterOperatorJSON("etcd", []configv1.ClusterOperatorStatusCondition{
		{Type: configv1.OperatorAvailable, Status: configv1.ConditionTrue},
		{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse},
		{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse},
	})
}

func fakeAPIServerPodListJSON(pods []corev1.Pod) []byte {
	podList := corev1.PodList{Items: pods}
	b, _ := json.Marshal(podList)
	return b
}

func healthyAPIServerPod(name, nodeName string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "openshift-kube-apiserver",
			Labels:    map[string]string{"app": "openshift-kube-apiserver"},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func healthyAPIServerPodsJSON() []byte {
	return fakeAPIServerPodListJSON([]corev1.Pod{
		healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
		healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
		healthyAPIServerPod("kube-apiserver-master-2", "master-2"),
	})
}

func allKubeChecksHealthyMock(k *mock_adminactions.MockKubeActions) {
	k.EXPECT().
		CheckAPIServerReadyz(gomock.Any()).
		Return(nil).
		AnyTimes()
	k.EXPECT().
		KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
		Return(healthyKubeAPIServerJSON(), nil).
		AnyTimes()
	k.EXPECT().
		KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
		Return(healthyAPIServerPodsJSON(), nil).
		AnyTimes()
	k.EXPECT().
		KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
		Return(healthyEtcdJSON(), nil).
		AnyTimes()
	k.EXPECT().
		KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
		Return(validServicePrincipalJSON(), nil).
		AnyTimes()
}

func TestPreResizeControlPlaneVMsValidation(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		vmSize         string
		fixture        func(f *testdatabase.Fixture)
		mocks          func(*test, *mock_adminactions.MockAzureActions)
		kubeMocks      func(*mock_adminactions.MockKubeActions)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "happy path - valid and available SKU",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					VMGetSKUs(gomock.Any(), []string{"Standard_D8s_v3"}).
					Return(map[string]*armcompute.ResourceSKU{
						"Standard_D8s_v3": {
							Name:         pointerutils.ToPtr("Standard_D8s_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"eastus"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("eastus"),
								},
							},
							Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
							Capabilities: []*armcompute.ResourceSKUCapabilities{},
						},
					}, nil)
			},
			kubeMocks:      allKubeChecksHealthyMock,
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"status":"passed"}` + "\n"),
		},
		{
			name:       "missing vmSize parameter",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions) {},
			kubeMocks:      allKubeChecksHealthyMock,
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : Pre-flight validation failed. Details: InvalidParameter: vmSize: The provided vmSize is empty.`,
		},
		{
			name:       "unsupported master VM size",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D2s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions) {},
			kubeMocks:      allKubeChecksHealthyMock,
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : Pre-flight validation failed. Details: InvalidParameter: : The provided vmSize 'Standard_D2s_v3' is unsupported for master.`,
		},
		{
			name:       "cluster not found",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
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
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "subscription doc not found",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})
			},
			mocks:          func(tt *test, a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf(`400: InvalidSubscriptionState: : Request is not allowed in unregistered subscription '%s'.`, mockSubID),
		},
		{
			name:       "SKU not available in region",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					VMGetSKUs(gomock.Any(), []string{"Standard_D8s_v3"}).
					Return(map[string]*armcompute.ResourceSKU{}, nil)
			},
			kubeMocks:      allKubeChecksHealthyMock,
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : Pre-flight validation failed. Details: InvalidParameter: vmSize: The selected SKU 'Standard_D8s_v3' is unavailable in region 'eastus'`,
		},
		{
			name:       "SKU restricted in subscription",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					VMGetSKUs(gomock.Any(), []string{"Standard_D8s_v3"}).
					Return(map[string]*armcompute.ResourceSKU{
						"Standard_D8s_v3": {
							Name:         pointerutils.ToPtr("Standard_D8s_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"eastus"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("eastus"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{
									Type: pointerutils.ToPtr(armcompute.ResourceSKURestrictionsTypeLocation),
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
									},
								},
							},
							Capabilities: []*armcompute.ResourceSKUCapabilities{},
						},
					}, nil)
			},
			kubeMocks:      allKubeChecksHealthyMock,
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : Pre-flight validation failed. Details: InvalidParameter: vmSize: The selected SKU 'Standard_D8s_v3' is restricted in region 'eastus' for selected subscription`,
		},
		{
			name:       "API server unreachable",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			vmSize:     "Standard_D8s_v3",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSizeStandardD8sV3,
							},
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
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {},
			kubeMocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					CheckAPIServerReadyz(gomock.Any()).
					Return(fmt.Errorf("connection refused"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: kube-apiserver: API server is reporting a non-ready status: connection refused`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.mocks(tt, a)

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			if tt.kubeMocks != nil {
				tt.kubeMocks(k)
			}

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Override quota validation to avoid creating real Azure clients in tests
			f.validateResizeQuota = quotaCheckDisabled

			go f.Run(ctx, nil, nil)

			url := fmt.Sprintf("https://server/admin%s/preresizevalidation", tt.resourceID)
			if tt.vmSize != "" {
				url += "?vmSize=" + tt.vmSize
			}

			resp, b, err := ti.request(http.MethodGet, url, nil, nil)
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

func TestCheckResizeComputeQuota(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name          string
		currentVMSize string
		vmSize        string
		mocks         func(*mock_compute.MockUsageClient)
		wantErr       string
	}

	for _, tt := range []*test{
		{
			// D8s_v3 (8 cores) → D16s_v3 (16 cores), same family.
			// Delta per node = 8, total = 8 × 3 = 24.  76 in use, limit 100 → 24 remaining = exact fit.
			name:          "same family upsize - enough quota for delta across all masters",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_D16s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardDSv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(76)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(76)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
		},
		{
			// D8s_v3 (8 cores) → D16s_v3 (16 cores), same family.
			// Delta per node = 8, total = 8 × 3 = 24.  77 in use, limit 100 → 23 remaining < 24.
			name:          "same family upsize - not enough quota for all masters",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_D16s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardDSv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(77)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(77)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: vmSize: Resource quota of standardDSv3Family exceeded. Maximum allowed: 100, Current in use: 77, Additional requested: 24.",
		},
		{
			name:          "same family downsize - no quota check needed",
			currentVMSize: "Standard_D16s_v3",
			vmSize:        "Standard_D8s_v3",
			mocks:         func(cuc *mock_compute.MockUsageClient) {},
		},
		{
			name:          "same family same size - no quota check needed",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_D8s_v3",
			mocks:         func(cuc *mock_compute.MockUsageClient) {},
		},
		{
			// D8s_v3 → E8s_v3, cross family.  Full new cores: 8 × 3 = 24.
			// Family: 76 in use, limit 100 → 24 remaining = exact fit.
			// Regional cores delta = (8 - 8) × 3 = 0, no check needed.
			name:          "cross family - full new cores checked for all masters",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_E8s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardESv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(76)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(100)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
		},
		{
			// D8s_v3 → E8s_v3, cross family.  Full new cores: 8 × 3 = 24.
			// 77 in use, limit 100 → 23 remaining < 24.
			name:          "cross family - not enough quota for all masters",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_E8s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardESv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(77)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(100)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: vmSize: Resource quota of standardESv3Family exceeded. Maximum allowed: 100, Current in use: 77, Additional requested: 24.",
		},
		{
			// D8s_v3 (8 cores) → E16s_v3 (16 cores), cross family upsize.
			// Family quota: plenty of room (50 in use, limit 200).
			// Regional cores delta = (16 - 8) × 3 = 24.  177 in use, limit 200 → 23 remaining < 24.
			name:          "cross family - regional cores quota exceeded",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_E16s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardESv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(50)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(177)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: vmSize: Resource quota of cores exceeded. Maximum allowed: 200, Current in use: 177, Additional requested: 24.",
		},
		{
			// D8s_v3 → E4s_v3, cross family downsize.
			// Family: full new cores = 4 × 3 = 12.
			// Regional cores delta = (4 - 8) × 3 = -12 → clamped to 0, no regional check.
			name:          "cross family downsize - regional cores not checked",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_E4s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardESv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(88)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("cores"),
							},
							CurrentValue: pointerutils.ToPtr(int32(199)),
							Limit:        pointerutils.ToPtr(int64(200)),
						},
					}, nil)
			},
		},
		{
			name:          "family not in usage list - no quota limit",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_D16s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardESv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(50)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
					}, nil)
			},
		},
		{
			name:          "unsupported new VM size",
			currentVMSize: "Standard_D8s_v3",
			vmSize:        "Standard_Nonexistent_v99",
			mocks:         func(cuc *mock_compute.MockUsageClient) {},
			wantErr:       "400: InvalidParameter: vmSize: The provided VM SKU 'Standard_Nonexistent_v99' is not supported.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			computeUsageClient := mock_compute.NewMockUsageClient(controller)
			tt.mocks(computeUsageClient)

			err := checkResizeComputeQuota(ctx, computeUsageClient, "eastus", tt.currentVMSize, tt.vmSize)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateVMSP(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "valid service principal",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return(validServicePrincipalJSON(), nil)
			},
		},
		{
			name: "invalid service principal",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return(fakeAROClusterJSON([]operatorv1.OperatorCondition{
						{
							Type:    arov1alpha1.ServicePrincipalValid,
							Status:  operatorv1.ConditionFalse,
							Message: "secret expired",
						},
					}), nil)
			},
			wantErr: "409: InvalidServicePrincipalCredentials: servicePrincipal: Cluster Service Principal is invalid: secret expired",
		},
		{
			name: "condition not found",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return(fakeAROClusterJSON([]operatorv1.OperatorCondition{}), nil)
			},
			wantErr: "409: InvalidServicePrincipalCredentials: servicePrincipal: ServicePrincipalValid condition not found on the ARO Cluster resource. The ARO operator may not have reconciled yet.",
		},
		{
			name: "KubeGet returns error",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return(nil, fmt.Errorf("connection refused"))
			},
			wantErr: "500: InternalServerError: servicePrincipal: Failed to retrieve ARO Cluster resource: connection refused",
		},
		{
			name: "KubeGet returns invalid JSON",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return([]byte(`{invalid`), nil)
			},
			wantErr: "500: InternalServerError: servicePrincipal: Failed to parse ARO Cluster resource: invalid character 'i' looking for beginning of object key string",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			k := mock_adminactions.NewMockKubeActions(controller)
			tt.mocks(k)

			err := validateClusterSP(ctx, k)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateAPIServerHealth(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "healthy kube-apiserver",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return(healthyKubeAPIServerJSON(), nil)
			},
		},
		{
			name: "kube-apiserver degraded",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return(fakeClusterOperatorJSON("kube-apiserver", []configv1.ClusterOperatorStatusCondition{
						{Type: configv1.OperatorAvailable, Status: configv1.ConditionTrue},
						{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse},
						{Type: configv1.OperatorDegraded, Status: configv1.ConditionTrue},
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver: kube-apiserver is not healthy: kube-apiserver Available=True, Progressing=False. Resize is not safe while the API server is degraded.",
		},
		{
			name: "kube-apiserver unavailable",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return(fakeClusterOperatorJSON("kube-apiserver", []configv1.ClusterOperatorStatusCondition{
						{Type: configv1.OperatorAvailable, Status: configv1.ConditionFalse},
						{Type: configv1.OperatorProgressing, Status: configv1.ConditionTrue},
						{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse},
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver: kube-apiserver is not healthy: kube-apiserver Available=False, Progressing=True. Resize is not safe while the API server is degraded.",
		},
		{
			name: "KubeGet returns error",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return(nil, fmt.Errorf("connection refused"))
			},
			wantErr: "500: InternalServerError: kube-apiserver: Failed to retrieve kube-apiserver ClusterOperator: connection refused",
		},
		{
			name: "KubeGet returns invalid JSON",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return([]byte(`{invalid`), nil)
			},
			wantErr: "500: InternalServerError: kube-apiserver: Failed to parse kube-apiserver ClusterOperator: invalid character 'i' looking for beginning of object key string",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			k := mock_adminactions.NewMockKubeActions(controller)
			tt.mocks(k)

			err := validateAPIServerHealth(ctx, k)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateEtcdHealth(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "healthy etcd",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(healthyEtcdJSON(), nil)
			},
		},
		{
			name: "etcd degraded",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(fakeClusterOperatorJSON("etcd", []configv1.ClusterOperatorStatusCondition{
						{Type: configv1.OperatorAvailable, Status: configv1.ConditionTrue},
						{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse},
						{Type: configv1.OperatorDegraded, Status: configv1.ConditionTrue},
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: etcd: etcd is not healthy: etcd Available=True, Progressing=False. Resize is not safe while etcd quorum is at risk.",
		},
		{
			name: "etcd unavailable",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(fakeClusterOperatorJSON("etcd", []configv1.ClusterOperatorStatusCondition{
						{Type: configv1.OperatorAvailable, Status: configv1.ConditionFalse},
						{Type: configv1.OperatorProgressing, Status: configv1.ConditionTrue},
						{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse},
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: etcd: etcd is not healthy: etcd Available=False, Progressing=True. Resize is not safe while etcd quorum is at risk.",
		},
		{
			name: "KubeGet returns error",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(nil, fmt.Errorf("connection refused"))
			},
			wantErr: "500: InternalServerError: etcd: Failed to retrieve etcd ClusterOperator: connection refused",
		},
		{
			name: "KubeGet returns invalid JSON",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return([]byte(`{invalid`), nil)
			},
			wantErr: "500: InternalServerError: etcd: Failed to parse etcd ClusterOperator: invalid character 'i' looking for beginning of object key string",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			k := mock_adminactions.NewMockKubeActions(controller)
			tt.mocks(k)

			err := validateEtcdHealth(ctx, k)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateAPIServerPods(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "all pods healthy",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(healthyAPIServerPodsJSON(), nil)
			},
		},
		{
			name: "KubeList returns error",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(nil, fmt.Errorf("connection refused"))
			},
			wantErr: "500: InternalServerError: kube-apiserver-pods: Failed to list pods in openshift-kube-apiserver namespace: connection refused",
		},
		{
			name: "KubeList returns invalid JSON",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return([]byte(`{invalid`), nil)
			},
			wantErr: "500: InternalServerError: kube-apiserver-pods: Failed to parse pod list: invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "only 2 apiserver pods",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON([]corev1.Pod{
						healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
						healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver-pods: Expected 3 kube-apiserver pods, found 2. Resize is not safe without full API server redundancy.",
		},
		{
			name: "4 apiserver pods",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON([]corev1.Pod{
						healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
						healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
						healthyAPIServerPod("kube-apiserver-master-2", "master-2"),
						healthyAPIServerPod("kube-apiserver-master-3", "master-3"),
					}), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver-pods: Expected 3 kube-apiserver pods, found 4. Resize is not safe without full API server redundancy.",
		},
		{
			name: "one pod not running",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				pods := []corev1.Pod{
					healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
					healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kube-apiserver-master-2",
							Namespace: "openshift-kube-apiserver",
							Labels:    map[string]string{"app": "openshift-kube-apiserver"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodPending,
						},
					},
				}
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON(pods), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver-pods: Unhealthy kube-apiserver pods: [kube-apiserver-master-2 (phase: Pending)]. Resize is not safe without full API server redundancy.",
		},
		{
			name: "one pod not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				pods := []corev1.Pod{
					healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
					healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kube-apiserver-master-2",
							Namespace: "openshift-kube-apiserver",
							Labels:    map[string]string{"app": "openshift-kube-apiserver"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{Type: corev1.PodReady, Status: corev1.ConditionFalse},
							},
						},
					},
				}
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON(pods), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver-pods: Unhealthy kube-apiserver pods: [kube-apiserver-master-2 (not ready)]. Resize is not safe without full API server redundancy.",
		},
		{
			name: "multiple unhealthy pods",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				pods := []corev1.Pod{
					healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kube-apiserver-master-1",
							Namespace: "openshift-kube-apiserver",
							Labels:    map[string]string{"app": "openshift-kube-apiserver"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodFailed,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kube-apiserver-master-2",
							Namespace: "openshift-kube-apiserver",
							Labels:    map[string]string{"app": "openshift-kube-apiserver"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{Type: corev1.PodReady, Status: corev1.ConditionFalse},
							},
						},
					},
				}
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON(pods), nil)
			},
			wantErr: "409: RequestNotAllowed: kube-apiserver-pods: Unhealthy kube-apiserver pods: [kube-apiserver-master-1 (phase: Failed) kube-apiserver-master-2 (not ready)]. Resize is not safe without full API server redundancy.",
		},
		{
			name: "filters non-apiserver pods",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				pods := []corev1.Pod{
					healthyAPIServerPod("kube-apiserver-master-0", "master-0"),
					healthyAPIServerPod("kube-apiserver-master-1", "master-1"),
					healthyAPIServerPod("kube-apiserver-master-2", "master-2"),
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "some-other-pod",
							Namespace: "openshift-kube-apiserver",
							Labels:    map[string]string{"app": "other-app"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
				}
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(fakeAPIServerPodListJSON(pods), nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			k := mock_adminactions.NewMockKubeActions(controller)
			tt.mocks(k)

			err := validateAPIServerPods(ctx, k)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
