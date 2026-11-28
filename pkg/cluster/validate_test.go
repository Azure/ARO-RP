package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
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
		wantErrs              []error

		// wantTarget is asserted when the case must return a CloudError.
		wantTarget string
		// wantCloudError is set for zone-validation failures, which must reach
		// the customer as a 400 InvalidParameter UserError.
		wantCloudError bool
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
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The selected control plane SKU 'Standard_D16as_v4' is available in only 2 of the 3 required availability zones in region 'eastus' (zones 1, 2, 3). Please select a SKU that is available in zones 1, 2, 3 in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			name:                 "zonal, worker unavailable",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected worker SKU 'Standard_D8as_v4' is available in only 2 of the 3 required availability zones in region 'eastus' (zones 1, 2, 3). Please select a SKU that is available in zones 1, 2, 3 in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.workerProfiles[0].VMSize",
		},
		{
			// Pins the boundary against a future < -> <= slip.
			name:                 "zonal, worker in a single zone",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected worker SKU 'Standard_D8as_v4' is available in only 1 of the 3 required availability zones in region 'eastus' (zones 1, 2, 3). Please select a SKU that is available in zones 1, 2, 3 in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.workerProfiles[0].VMSize",
		},
		{
			name:                 "mixed, control plane non-zonal",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The control plane SKU 'Standard_D16as_v4' is not available in availability zones in region 'eastus', but the worker SKU 'Standard_D8as_v4' is. Zonal and non-zonal SKUs cannot be mixed in a cluster. Please select SKUs that are either both zonal or both non-zonal in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			name:                 "mixed, worker non-zonal",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.workerProfiles[0].VMSize: The worker SKU 'Standard_D8as_v4' is not available in availability zones in region 'eastus', but the control plane SKU 'Standard_D16as_v4' is. Zonal and non-zonal SKUs cannot be mixed in a cluster. Please select SKUs that are either both zonal or both non-zonal in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.workerProfiles[0].VMSize",
		},
		{
			// With expanded AZs off, FilterZones strips zone 4 and leaves the
			// control plane with only 2 usable zones.
			name:                 "expanded AZs off collapses control plane to 2 zones",
			controlPlaneSkuZones: []string{"1", "2", "4"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The selected control plane SKU 'Standard_D16as_v4' is available in only 2 of the 3 required availability zones in region 'eastus' (zones 1, 2, 3). Please select a SKU that is available in zones 1, 2, 3 in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			// Expanded AZs on: only the zones Azure actually offers are counted,
			// so the message compares against the requirement directly.
			name:                 "expanded AZs on, control plane unavailable",
			expandedAZs:          true,
			controlPlaneSkuZones: []string{"1", "2"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The selected control plane SKU 'Standard_D16as_v4' is available in only 2 of the 3 required availability zones in region 'eastus'. Please select a SKU that is available in at least 3 availability zones in this region.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			// A SKU only in zones ARO does not support is zonal, so this must
			// not be reported as a mix of zonal and non-zonal SKUs.
			name:                 "control plane only in unsupported zones, expanded AZs off",
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The selected control plane SKU 'Standard_D16as_v4' is available in 3 availability zone(s) in region 'eastus', but none of them are zones 1, 2, 3, which ARO requires for zonal cluster creation.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			name:                 "worker only in unsupported zones, expanded AZs off",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"4", "5", "6"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected worker SKU 'Standard_D8as_v4' is available in 3 availability zone(s) in region 'eastus', but none of them are zones 1, 2, 3, which ARO requires for zonal cluster creation.")},
			wantCloudError:       true,
			wantTarget:           "properties.workerProfiles[0].VMSize",
		},
		{
			// Asymmetric: zonal control plane restricted to unsupported zones,
			// genuinely non-zonal worker. Must be an insufficient-zones error on
			// the control plane, not the unactionable mixed-SKU message.
			name:                 "control plane only in unsupported zones, non-zonal workers",
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.masterProfile.VMSize: The selected control plane SKU 'Standard_D16as_v4' is available in 3 availability zone(s) in region 'eastus', but none of them are zones 1, 2, 3, which ARO requires for zonal cluster creation.")},
			wantCloudError:       true,
			wantTarget:           "properties.masterProfile.VMSize",
		},
		{
			name:                 "worker only in unsupported zones, non-zonal control plane",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"4", "5", "6"},
			wantErrs:             []error{errors.New("400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected worker SKU 'Standard_D8as_v4' is available in 3 availability zone(s) in region 'eastus', but none of them are zones 1, 2, 3, which ARO requires for zonal cluster creation.")},
			wantCloudError:       true,
			wantTarget:           "properties.workerProfiles[0].VMSize",
		},
		{
			// With expanded AZs on the same SKUs are perfectly usable.
			name:                 "unsupported zones are usable when expanded AZs are on",
			expandedAZs:          true,
			controlPlaneSkuZones: []string{"4", "5", "6"},
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
					Zones: []string{"1", "2", "3", "4", "5", "6"},
				},
			},
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
					Name:      new(workerProfileSku),
					Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
						{Zones: pointerutils.ToSlicePtr(tt.workerSkuZones)},
					}),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
					ResourceType: new("virtualMachines"),
				},
				{
					Name:      &controlPlaneSku,
					Locations: pointerutils.ToSlicePtr([]string{"eastus"}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
						{Zones: pointerutils.ToSlicePtr(tt.controlPlaneSkuZones)},
					}),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
					ResourceType: new("virtualMachines"),
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
				List(gomock.Any(), "location eq eastus", false).
				Return(func(yield func(armcompute.ResourceSKU, error) bool) {
					if tt.resourceSkusClientErr != nil {
						yield(armcompute.ResourceSKU{}, tt.resourceSkusClientErr)
						return
					}
					for _, v := range skus {
						if !yield(*v, nil) {
							return
						}
					}
				})

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
				armResourceSKUs: resourceSkusClient,
				db:              openShiftClustersDatabase,
			}

			fixture.AddOpenShiftClusterDocuments(m.doc)
			fixture.Create()

			err := m.validateZones(ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.wantErrs)

			if tt.wantCloudError {
				var cloudErr *api.CloudError
				if !errors.As(err, &cloudErr) {
					t.Fatalf("expected *api.CloudError, got %T", err)
				}
				if cloudErr.StatusCode != http.StatusBadRequest {
					t.Errorf("got status %d, want %d", cloudErr.StatusCode, http.StatusBadRequest)
				}
				if cloudErr.Code != api.CloudErrorCodeInvalidParameter {
					t.Errorf("got code %q, want %q", cloudErr.Code, api.CloudErrorCodeInvalidParameter)
				}
				if cloudErr.Target != tt.wantTarget {
					t.Errorf("got target %q, want %q", cloudErr.Target, tt.wantTarget)
				}
				if got := utillog.MapStatusCodeToResultType(cloudErr.StatusCode); got != utillog.UserErrorResultType {
					t.Errorf("got resultType %q, want %q", got, utillog.UserErrorResultType)
				}
			}
			if len(tt.wantErrs) == 0 {
				for _, err := range checker.CheckOpenShiftClusters(openShiftClustersClient) {
					t.Error(err)
				}
			}
		})
	}
}
