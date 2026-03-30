package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	sdkcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
)

const (
	standardDisk          = "StandardSSD_LRS"
	premiumDisk           = "Premium_LRS"
	premiumDiskCapability = "PremiumIO"
)

var ErrListVMResourceSKUs = errors.New("failure listing resource SKUs")

// Zones returns zone information for the resource SKU
func Zones(sku *sdkcompute.ResourceSKU) []string {
	if len(sku.LocationInfo) == 0 ||
		sku.LocationInfo[0] == nil ||
		(sku.LocationInfo)[0].Zones == nil {
		return nil
	}

	r := []string{}
	for _, x := range sku.LocationInfo[0].Zones {
		r = append(r, *x)
	}
	return r
}

// HasCapability checks whether given resource SKU has specific capability
func HasCapability(sku *sdkcompute.ResourceSKU, capabilityName string) bool {
	if sku.Capabilities == nil {
		return false
	}

	for _, c := range sku.Capabilities {
		if *c.Name == capabilityName {
			return *c.Value == "True"
		}
	}

	return false
}

// IsRestricted checks whether given resource SKU is restricted in a given location
func IsRestricted(sku *sdkcompute.ResourceSKU, location string) bool {
	for _, restriction := range sku.Restrictions {
		if restriction != nil && restriction.RestrictionInfo != nil {
			for _, restrictedLocation := range restriction.RestrictionInfo.Locations {
				if restrictedLocation != nil && strings.EqualFold(*restrictedLocation, location) {
					return true
				}
			}
		}
	}

	return false
}

// SupportedOSDisk returns the type of OSDisk for the given resource. Most VMs will use Premium disks but some SKUs only support Standard SSDs
func SupportedOSDisk(vmSku *sdkcompute.ResourceSKU) string {
	if HasCapability(vmSku, premiumDiskCapability) {
		return premiumDisk
	}
	return standardDisk
}

func SelectVMSkusInCurrentRegion(ctx context.Context, resourceSkusClient armcompute.ResourceSKUsClient, location string, skuNames []string) (map[string]*sdkcompute.ResourceSKU, error) {
	// Sort and compact so that we only have one instance of each SKU in the list
	slices.Sort(skuNames)
	skuNames = slices.Compact(skuNames)

	vmskus := map[string]*sdkcompute.ResourceSKU{}
	filter := fmt.Sprintf("location eq %s", location)
	skusIter := resourceSkusClient.List(ctx, filter, false)

	for sku, err := range skusIter {
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListVMResourceSKUs, err)
		}

		if sku == nil || sku.ResourceType == nil || sku.Name == nil {
			continue
		}

		// We only care about VMs and ones with locations/locationinfo
		if *sku.ResourceType != "virtualMachines" || len(sku.Locations) == 0 || len(sku.LocationInfo) == 0 {
			continue
		}

		// Make sure it's actually in our location
		if !slices.ContainsFunc(sku.Locations, func(s *string) bool { return s != nil && strings.EqualFold(*s, location) }) {
			continue
		}

		if slices.Contains(skuNames, *sku.Name) {
			vmskus[*sku.Name] = sku
		}

		// If we've already found all the SKUs we want, exit
		if len(vmskus) == len(skuNames) {
			break
		}
	}

	return vmskus, nil
}

func ListUnrestrictedVMSkusInCurrentRegion(ctx context.Context, resourceSkusClient armcompute.ResourceSKUsClient, location string) ([]string, error) {
	vmskus := []string{}
	filter := fmt.Sprintf("location eq %s", location)
	skusIter := resourceSkusClient.List(ctx, filter, false)

	for sku, err := range skusIter {
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListVMResourceSKUs, err)
		}

		if sku == nil || sku.ResourceType == nil || sku.Name == nil {
			continue
		}

		// We only care about VMs and ones with locations/locationinfo
		if *sku.ResourceType != "virtualMachines" || len(sku.Locations) == 0 || len(sku.LocationInfo) == 0 {
			continue
		}

		// Make sure it's actually in our location
		if !slices.ContainsFunc(sku.Locations, func(s *string) bool { return s != nil && strings.EqualFold(*s, location) }) {
			continue
		}

		if !IsRestricted(sku, location) {
			vmskus = append(vmskus, *sku.Name)
		}
	}

	slices.Sort(vmskus)
	return slices.Compact(vmskus), nil
}
