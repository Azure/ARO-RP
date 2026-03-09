package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

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
					VMSizeList(gomock.Any()).
					Return([]*armcompute.ResourceSKU{
						{
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
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`"All pre-flight checks passed"` + "\n"),
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
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: vmSize: The provided vmSize is empty.`,
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
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : The provided vmSize 'Standard_D2s_v3' is unsupported for master.`,
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
					VMSizeList(gomock.Any()).
					Return([]*armcompute.ResourceSKU{
						{
							Name:         pointerutils.ToPtr("Standard_D16s_v3"),
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
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: vmSize: The selected SKU 'Standard_D8s_v3' is unavailable in region 'eastus'`,
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
					VMSizeList(gomock.Any()).
					Return([]*armcompute.ResourceSKU{
						{
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
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: vmSize: The selected SKU 'Standard_D8s_v3' is restricted in region 'eastus' for selected subscription`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.mocks(tt, a)

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
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
		name    string
		vmSize  string
		mocks   func(*mock_compute.MockUsageClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name:   "enough quota available",
			vmSize: "Standard_D8s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardDSv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(10)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
					}, nil)
			},
		},
		{
			name:   "exact quota available",
			vmSize: "Standard_D8s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardDSv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(92)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
					}, nil)
			},
		},
		{
			name:   "not enough quota",
			vmSize: "Standard_D8s_v3",
			mocks: func(cuc *mock_compute.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "eastus").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: pointerutils.ToPtr("standardDSv3Family"),
							},
							CurrentValue: pointerutils.ToPtr(int32(93)),
							Limit:        pointerutils.ToPtr(int64(100)),
						},
					}, nil)
			},
			wantErr: "400: ResourceQuotaExceeded: vmSize: Resource quota of standardDSv3Family exceeded. Maximum allowed: 100, Current in use: 93, Additional requested: 8.",
		},
		{
			name:   "family not in usage list - no quota limit",
			vmSize: "Standard_D8s_v3",
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
			name:    "unsupported VM size",
			vmSize:  "Standard_Nonexistent_v99",
			mocks:   func(cuc *mock_compute.MockUsageClient) {},
			wantErr: "400: InvalidParameter: vmSize: The provided VM SKU 'Standard_Nonexistent_v99' is not supported.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			computeUsageClient := mock_compute.NewMockUsageClient(controller)
			tt.mocks(computeUsageClient)

			err := checkResizeComputeQuota(ctx, computeUsageClient, "eastus", tt.vmSize)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
