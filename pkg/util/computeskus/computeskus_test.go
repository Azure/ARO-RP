package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
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

func TestFilterVmSizes(t *testing.T) {
	for _, tt := range []struct {
		name             string
		providedLocation string
		resourceType     string
		skuLocation      []string
		skuRestrictions  armcompute.ResourceSKURestrictions
		skuLocationInfo  []armcompute.ResourceSKULocationInfo
		skuCapabilities  string
		wantResult       map[string]*armcompute.ResourceSKU
	}{
		{
			name:             "resource type is a virtual machine",
			providedLocation: "eastus",
			resourceType:     "virtualMachines",
			skuRestrictions:  armcompute.ResourceSKURestrictions{ReasonCode: pointerutils.ToPtr(armcompute.ResourceSKURestrictionsReasonCodeNotAvailableForSubscription)},
			skuLocation:      []string{"eastus"},
			skuLocationInfo:  []armcompute.ResourceSKULocationInfo{{Zones: pointerutils.ToSlicePtr([]string{"eastus-2"})}},
			skuCapabilities:  "some-capability",

			wantResult: map[string]*armcompute.ResourceSKU{
				"Fake_Sku": {
					Name: pointerutils.ToPtr("Fake_Sku"),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{{
						ReasonCode: pointerutils.ToPtr(armcompute.ResourceSKURestrictionsReasonCodeNotAvailableForSubscription)}}),
					LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{{
						Zones: pointerutils.ToSlicePtr([]string{"eastus-2"})},
					}),
					Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{{
						Name: pointerutils.ToPtr("some-capability"),
					}}),
				},
			},
		},
		{
			name:             "resource type not a virtual machine",
			providedLocation: "eastus",
			resourceType:     "disk",
			skuLocation:      []string{"eastus"},
			skuLocationInfo:  []armcompute.ResourceSKULocationInfo{{Zones: pointerutils.ToSlicePtr([]string{"eastus-2"})}},
			wantResult:       map[string]*armcompute.ResourceSKU{},
		},
		{
			name:             "sku Location doesn't match provided location",
			providedLocation: "mars",
			resourceType:     "virtualMachines",
			skuLocation:      []string{"eastus"},
			skuLocationInfo:  []armcompute.ResourceSKULocationInfo{{Zones: pointerutils.ToSlicePtr([]string{"eastus-2"})}},
			wantResult:       map[string]*armcompute.ResourceSKU{},
		},
		{
			name:             "sku Location has length of 0",
			providedLocation: "eastus",
			resourceType:     "virtualMachines",
			skuLocation:      []string{},
			skuLocationInfo:  []armcompute.ResourceSKULocationInfo{{Zones: pointerutils.ToSlicePtr([]string{"eastus-2"})}},
			wantResult:       map[string]*armcompute.ResourceSKU{},
		},
		{
			name:             "sku LocationInfo has length of 0",
			providedLocation: "eastus",
			resourceType:     "virtualMachines",
			skuLocation:      []string{"eastus"},
			skuLocationInfo:  []armcompute.ResourceSKULocationInfo{},
			wantResult:       map[string]*armcompute.ResourceSKU{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sku := []*armcompute.ResourceSKU{
				{
					Name: pointerutils.ToPtr("Fake_Sku"),
					Capabilities: pointerutils.ToSlicePtr([]armcompute.ResourceSKUCapabilities{
						{
							Name: pointerutils.ToPtr(tt.skuCapabilities),
						},
					}),
					Locations:    pointerutils.ToSlicePtr(tt.skuLocation),
					Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{tt.skuRestrictions}),
					LocationInfo: pointerutils.ToSlicePtr(tt.skuLocationInfo),
					ResourceType: pointerutils.ToPtr(tt.resourceType),
				},
			}

			result := FilterVMSizes(sku, tt.providedLocation)

			if !reflect.DeepEqual(result, tt.wantResult) {
				t.Error(cmp.Diff(result, tt.wantResult))
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
