package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateZones(t *testing.T) {
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

	controlPlaneSku := string(vms.VMSizeStandardD16asV4)
	workerProfileSku := string(vms.VMSizeStandardD8asV4)

	type test struct {
		name string

		controlPlaneSkuZones []string
		workerSkuZones       []string
		expandedAZs          bool

		doc                   api.OpenShiftCluster
		resourceSkusClientErr error
		wantErrs              []error
	}

	for _, tt := range []*test{
		{
			name:                 "non-zonal",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{},
			doc: api.OpenShiftCluster{
				ID:       key,
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						VMSize: vms.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: vms.VMSizeStandardD8asV4,
						},
					},
					Zones: []string{},
				},
			},
		},
		{
			name:                 "zonal, all available",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2", "3"},
			doc: api.OpenShiftCluster{
				ID:       key,
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						VMSize: vms.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: vms.VMSizeStandardD8asV4,
						},
					},
					Zones: []string{"1", "2", "3"},
				},
			},
		},
		{
			name:                 "zonal, all available, expanded AZs off",
			controlPlaneSkuZones: []string{"1", "2", "3", "4"},
			workerSkuZones:       []string{"1", "2", "3", "4"},
			doc: api.OpenShiftCluster{
				ID:       key,
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						VMSize: vms.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: vms.VMSizeStandardD8asV4,
						},
					},
					Zones: []string{"1", "2", "3"},
				},
			},
		},
		{
			name:                 "zonal, all available, expanded AZs on",
			controlPlaneSkuZones: []string{"1", "2", "3", "4"},
			workerSkuZones:       []string{"1", "2", "3", "4"},
			expandedAZs:          true,
			doc: api.OpenShiftCluster{
				ID:       key,
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						VMSize: vms.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: vms.VMSizeStandardD8asV4,
						},
					},
					Zones: []string{"1", "2", "3", "4"},
				},
			},
		},
		{
			name:                 "zonal, control plane unavailable",
			controlPlaneSkuZones: []string{"1", "2"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErrs:             []error{errors.New("control plane SKU 'Standard_D16as_v4' only available in 2 zones, need 3")},
		},
		{
			name:                  "error from resourceskus",
			resourceSkusClientErr: errTestSKUFetchError,
			controlPlaneSkuZones:  []string{"1", "2"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantErrs:              []error{computeskus.ErrListVMResourceSKUs, errTestSKUFetchError},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)

			skus := []*armcompute.ResourceSKU{
				{
					Name:      pointerutils.ToPtr(workerProfileSku),
					Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
						{Zones: pointerutils.ToSlicePtr(tt.workerSkuZones)},
					}),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
				},
				{
					Name:      &controlPlaneSku,
					Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
						{Zones: pointerutils.ToSlicePtr(tt.controlPlaneSkuZones)},
					}),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
				},
			}

			mockEnv := mock_env.NewMockInterface(controller)

			if tt.resourceSkusClientErr == nil {
				mockEnv.EXPECT().FeatureIsSet(env.FeatureEnableClusterExpandedAvailabilityZones).Return(tt.expandedAZs)
			}

			openShiftClustersDatabase, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)

			checker := testdatabase.NewChecker()
			checker.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key:              strings.ToLower(key),
				OpenShiftCluster: &tt.doc,
			})

			resourceSkusClient := mock_armcompute.NewMockResourceSKUsClient(controller)
			resourceSkusClient.EXPECT().
				List(gomock.Any(), fmt.Sprintf("location eq %v", "eastus"), false).
				Return(skus, tt.resourceSkusClientErr)

			m := &manager{
				env: mockEnv,
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							WorkerProfiles: []api.WorkerProfile{
								{
									VMSize: vms.VMSize(workerProfileSku),
								},
							},
							MasterProfile: api.MasterProfile{
								VMSize: vms.VMSize(controlPlaneSku),
							},
						},
					},
				},
				armResourceSKUs: resourceSkusClient,
				db:              openShiftClustersDatabase,
			}

			fixture.AddOpenShiftClusterDocuments(m.doc)
			fixture.Create()

			err := m.validateZones(ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.wantErrs)
			if len(tt.wantErrs) == 0 {
				for _, err := range checker.CheckOpenShiftClusters(openShiftClustersClient) {
					t.Error(err)
				}
			}
		})
	}
}
