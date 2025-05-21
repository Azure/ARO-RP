package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
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

		doc                   api.OpenShiftCluster
		resourceSkusClientErr error
		wantErr               string
	}

	for _, tt := range []*test{
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
						Zones:  []string{"1", "2", "3"},
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSizeStandardD8asV4,
							Zones:  []string{"1", "2", "3"},
						},
					},
					NetworkProfile: api.NetworkProfile{
						LoadBalancerProfile: &api.LoadBalancerProfile{
							OutboundIPAvailabilityZones: []string{"1", "2", "3"},
						},
					},
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
			skus := []mgmtcompute.ResourceSku{
				{
					Name:      &workerProfileSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &tt.workerSkuZones},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					ResourceType: to.StringPtr("virtualMachines"),
				},
				{
					Name:      &controlPlaneSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &tt.controlPlaneSkuZones},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					ResourceType: to.StringPtr("virtualMachines"),
				},
			}

			controller := gomock.NewController(t)
			defer controller.Finish()
			ctx := context.Background()

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
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{},
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
