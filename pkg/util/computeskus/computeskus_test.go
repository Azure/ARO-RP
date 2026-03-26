package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"maps"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestZones(t *testing.T) {
	for _, tt := range []struct {
		name      string
		sku       *armcompute.ResourceSKU
		wantZones []string
	}{
		{
			name: "sku with location info present",
			sku: &armcompute.ResourceSKU{
				LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
					{Zones: pointerutils.ToSlicePtr([]string{"1", "2", "3"})},
				}),
			},
			wantZones: []string{"1", "2", "3"},
		},
		{
			name: "sku with location info present, but zones field is nil",
			sku: &armcompute.ResourceSKU{
				LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
					{Zones: nil},
				}),
			},
		},
		{
			name: "sku with location info present, but empty",
			sku: &armcompute.ResourceSKU{
				LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{}),
			},
		},
		{
			name: "sku with location info missing",
			sku:  &armcompute.ResourceSKU{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zones := Zones(tt.sku)

			if !reflect.DeepEqual(tt.wantZones, zones) {
				t.Error(cmp.Diff(tt.wantZones, zones))
			}
		})
	}
}

func TestHasCapability(t *testing.T) {
	fakeCapabilityName := "fakeCapability"

	for _, tt := range []struct {
		name       string
		sku        *armcompute.ResourceSKU
		wantResult bool
	}{
		{
			name: "sku explicitly supports capability",
			sku: &armcompute.ResourceSKU{
				Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{
					{Name: &fakeCapabilityName, Value: pointerutils.ToPtr("True")},
				}),
			},
			wantResult: true,
		},
		{
			name: "sku explicitly does not support capability",
			sku: &armcompute.ResourceSKU{
				Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{
					{Name: &fakeCapabilityName, Value: pointerutils.ToPtr("False")},
				}),
			},
		},
		{
			name: "sku implicitly does not support capability because it is missing from the list",
			sku: &armcompute.ResourceSKU{
				Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{}),
			},
		},
		{
			name: "sku implicitly does not support capability, because capabilities info missing",
			sku:  &armcompute.ResourceSKU{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := HasCapability(tt.sku, fakeCapabilityName)

			if result != tt.wantResult {
				t.Error(result)
			}
		})
	}
}

func TestIsRestricted(t *testing.T) {
	for _, tt := range []struct {
		name       string
		location   string
		vmsize     string
		sku        map[string]*armcompute.ResourceSKU
		wantResult bool
	}{
		{
			name:     "sku is restricted in one location",
			location: "eastus",
			vmsize:   "Standard_Sku_1",
			sku: map[string]*armcompute.ResourceSKU{
				"Standard_Sku_1": {Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{
					{
						RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{Locations: pointerutils.ToSlicePtr([]string{"eastus"})},
					},
				})},
			},
			wantResult: true,
		},
		{
			name:     "sku is restricted in multiple locations",
			location: "eastus",
			vmsize:   "Standard_Sku_1",
			sku: map[string]*armcompute.ResourceSKU{
				"Standard_Sku_1": {Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{
					{
						RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{Locations: pointerutils.ToSlicePtr([]string{
							"eastus",
							"eastus2",
						})},
					},
				})},
			},
			wantResult: true,
		},
		{
			name:     "sku is not restricted",
			location: "eastus",
			vmsize:   "Standard_Sku_2",
			sku: map[string]*armcompute.ResourceSKU{
				"Standard_Sku_2": {Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{
					{
						RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{Locations: pointerutils.ToSlicePtr([]string{""})},
					},
				})},
			},
			wantResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRestricted(tt.sku[tt.vmsize], tt.location)

			if result != tt.wantResult {
				t.Error(result)
			}
		})
	}
}

func TestSupportedOSDisk(t *testing.T) {
	for _, tt := range []struct {
		name                string
		vmSku               string
		supportsPremiumDisk string
		wantOSDisk          string
	}{
		{
			name:                "Premium disk supported on VMSize",
			vmSku:               "premium_disk_supported",
			supportsPremiumDisk: "True",
			wantOSDisk:          premiumDisk,
		},
		{
			name:                "Premium disk not supported on VMSize",
			vmSku:               "premium_disk_not_supported",
			supportsPremiumDisk: "False",
			wantOSDisk:          standardDisk,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resourceSku := &armcompute.ResourceSKU{
				Name: &tt.vmSku,
				Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{
					{
						Name:  pointerutils.ToPtr(premiumDiskCapability),
						Value: &tt.supportsPremiumDisk,
					},
				}),
			}

			result := SupportedOSDisk(resourceSku)
			if result != tt.wantOSDisk {
				t.Errorf("got %v but want %v", result, tt.wantOSDisk)
			}
		})
	}
}

func TestSelectVMSkusInCurrentRegion(t *testing.T) {
	for _, tt := range []struct {
		name      string
		vmSkus    []string
		mocks     func(*mock_armcompute.MockResourceSKUsClient)
		desired   map[string]*armcompute.ResourceSKU
		wantError string
	}{
		{
			name:   "happypath",
			vmSkus: []string{"bigmachine_v1", "smallmachine_v4", "smallmachine_v5"},
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Not actually in our region
						{
							Name:         pointerutils.ToPtr("smallmachine_v5"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus1"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus1"),
								},
							},
						}: nil,
						// Nil region
						{
							Name:         pointerutils.ToPtr("smallmachine_v6"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    []*string{nil},
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Machine that has no locations/locationinfo
						{
							Name:         pointerutils.ToPtr("smallmachine_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    nil,
							LocationInfo: nil,
						}: nil,
						// Actually an availabilitySet
						{
							Name:         pointerutils.ToPtr("Classic"),
							ResourceType: pointerutils.ToPtr("availabilitySets"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
					}),
				)
			},
			desired: map[string]*armcompute.ResourceSKU{
				"bigmachine_v1": {
					Name:         pointerutils.ToPtr("bigmachine_v1"),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
					Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
					LocationInfo: []*armcompute.ResourceSKULocationInfo{
						{
							Location: pointerutils.ToPtr("northus2"),
						},
					},
				},
				"smallmachine_v4": {
					Name:         pointerutils.ToPtr("smallmachine_v4"),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
					Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
					LocationInfo: []*armcompute.ResourceSKULocationInfo{
						{
							Location: pointerutils.ToPtr("northus2"),
						},
					},
				},
			},
		},
		{
			name:   "duplicate skus in vmskus",
			vmSkus: []string{"bigmachine_v1", "bigmachine_v1", "smallmachine_v4"},
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Machine that has no locations/locationinfo
						{
							Name:         pointerutils.ToPtr("smallmachine_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    nil,
							LocationInfo: nil,
						}: nil,
						// Actually an availabilitySet
						{
							Name:         pointerutils.ToPtr("Classic"),
							ResourceType: pointerutils.ToPtr("availabilitySets"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
					}),
				)
			},
			desired: map[string]*armcompute.ResourceSKU{
				"bigmachine_v1": {
					Name:         pointerutils.ToPtr("bigmachine_v1"),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
					Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
					LocationInfo: []*armcompute.ResourceSKULocationInfo{
						{
							Location: pointerutils.ToPtr("northus2"),
						},
					},
				},
				"smallmachine_v4": {
					Name:         pointerutils.ToPtr("smallmachine_v4"),
					ResourceType: pointerutils.ToPtr("virtualMachines"),
					Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
					LocationInfo: []*armcompute.ResourceSKULocationInfo{
						{
							Location: pointerutils.ToPtr("northus2"),
						},
					},
				},
			},
		},
		{
			name:   "error in pagination",
			vmSkus: []string{"bigmachine_v5"},
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						nil: errors.New("this is an error"),
					}),
				)
			},
			wantError: "this is an error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			mock_skus := mock_armcompute.NewMockResourceSKUsClient(controller)
			tt.mocks(mock_skus)

			r, err := SelectVMSkusInCurrentRegion(t.Context(), mock_skus, "northus2", tt.vmSkus)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.desired, r)
			}
		})
	}
}

func TestListUnrestrictedSKUNames(t *testing.T) {
	for _, tt := range []struct {
		name      string
		mocks     func(*mock_armcompute.MockResourceSKUsClient)
		desired   []string
		wantError string
	}{
		{
			name: "happypath",
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Capitalisation of region
						{
							Name:         pointerutils.ToPtr("smallmachine_v10"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"Northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("Northus2"),
								},
							},
						}: nil,
						// Duplicated struct, in case we get two
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{},
						}: nil,
						// Not actually in our region
						{
							Name:         pointerutils.ToPtr("smallmachine_v9"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus1"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus1"),
								},
							},
						}: nil,
						// Nil region
						{
							Name:         pointerutils.ToPtr("smallmachine_v11"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    []*string{nil},
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Restricted in this region
						{
							Name:         pointerutils.ToPtr("smallmachine_v2"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"somewhereelse"}),
									},
								},
								{
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"northus2"}),
									},
								},
							},
						}: nil,
						// Restricted in this region, equal fold
						{
							Name:         pointerutils.ToPtr("smallmachine_v20"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"Northus2"}),
									},
								},
							},
						}: nil,
						// Nil restriction info
						{
							Name:         pointerutils.ToPtr("smallmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{},
							},
						}: nil,
						// Machine that has no locations/locationinfo
						{
							Name:         pointerutils.ToPtr("smallmachine_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    nil,
							LocationInfo: nil,
						}: nil,
						// Actually an availabilitySet
						{
							Name:         pointerutils.ToPtr("Classic"),
							ResourceType: pointerutils.ToPtr("availabilitySets"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
					}),
				)
			},
			desired: []string{"bigmachine_v1", "smallmachine_v1", "smallmachine_v10", "smallmachine_v4"},
		},
		{
			name: "duplicate VM structs don't lead to duplicated names",
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
								{},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Restricted in this region
						{
							Name:         pointerutils.ToPtr("smallmachine_v2"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"northus2"}),
									},
								},
							},
						}: nil,
						// Machine that has no locations/locationinfo
						{
							Name:         pointerutils.ToPtr("smallmachine_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    nil,
							LocationInfo: nil,
						}: nil,
						// Actually an availabilitySet
						{
							Name:         pointerutils.ToPtr("Classic"),
							ResourceType: pointerutils.ToPtr("availabilitySets"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
					}),
				)
			},
			desired: []string{"bigmachine_v1", "smallmachine_v4"},
		},
		{
			name: "error in pagination",
			mocks: func(mrsc *mock_armcompute.MockResourceSKUsClient) {
				mrsc.EXPECT().List(gomock.Any(), "location eq northus2", false).Return(
					maps.All(map[*armcompute.ResourceSKU]error{
						{
							Name:         pointerutils.ToPtr("bigmachine_v1"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						{
							Name:         pointerutils.ToPtr("smallmachine_v4"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
						}: nil,
						// Restricted in this region
						{
							Name:         pointerutils.ToPtr("smallmachine_v2"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"northus2"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("northus2"),
								},
							},
							Restrictions: []*armcompute.ResourceSKURestrictions{
								{
									RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
										Locations: pointerutils.ToSlicePtr([]string{"northus2"}),
									},
								},
							},
						}: nil,
						nil: errors.New("this is an error"),
					}),
				)
			},
			wantError: "this is an error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			mock_skus := mock_armcompute.NewMockResourceSKUsClient(controller)
			tt.mocks(mock_skus)

			r, err := ListUnrestrictedVMSkusInCurrentRegion(t.Context(), mock_skus, "northus2")
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.desired, r)
			}
		})
	}
}
