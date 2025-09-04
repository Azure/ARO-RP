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

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateZones(t *testing.T) {
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

	controlPlaneSku := string(api.VMSizeStandardD16asV4)
	workerProfileSku := string(api.VMSizeStandardD8asV4)

	type test struct {
		name string

		controlPlaneSkuZones []string
		workerSkuZones       []string
		expandedAZs          bool

		doc                   api.OpenShiftCluster
		resourceSkusClientErr error
		wantErr               string
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
						VMSize: api.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSizeStandardD8asV4,
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
						VMSize: api.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSizeStandardD8asV4,
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
						VMSize: api.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSizeStandardD8asV4,
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
						VMSize: api.VMSizeStandardD16asV4,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSizeStandardD8asV4,
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
			wantErr:              "control plane SKU 'Standard_D16as_v4' only available in 2 zones, need 3",
		},
		{
			name:                  "error from resourceskus",
			resourceSkusClientErr: errors.New("error time :)"),
			controlPlaneSkuZones:  []string{"1", "2"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantErr:               "failure listing resource SKUs: error time :)",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)

			skus := []mgmtcompute.ResourceSku{
				{
					Name:      &workerProfileSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &tt.workerSkuZones},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					ResourceType: pointerutils.ToPtr("virtualMachines"),
				},
				{
					Name:      &controlPlaneSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &tt.controlPlaneSkuZones},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
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

			resourceSkusClient := mock_compute.NewMockResourceSkusClient(controller)
			resourceSkusClient.EXPECT().
				List(gomock.Any(), fmt.Sprintf("location eq %v", "eastus")).
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
									VMSize: api.VMSize(workerProfileSku),
								},
							},
							MasterProfile: api.MasterProfile{
								VMSize: api.VMSize(controlPlaneSku),
							},
						},
					},
				},
				resourceSkus: resourceSkusClient,
				db:           openShiftClustersDatabase,
			}

			fixture.AddOpenShiftClusterDocuments(m.doc)
			fixture.Create()

			err := m.validateZones(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantErr == "" {
				for _, err := range checker.CheckOpenShiftClusters(openShiftClustersClient) {
					t.Error(err)
				}
			}
		})
	}
}
