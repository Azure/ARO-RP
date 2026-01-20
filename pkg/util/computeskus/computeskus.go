package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
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
		for _, restrictedLocation := range restriction.RestrictionInfo.Locations {
			if *restrictedLocation == location {
				return true
			}
		}
	}

	return false
}

// FilterVMSizes filters resource SKU by location and returns only virtual machines, their names, restrictions, location info, and capabilities.
func FilterVMSizes(skus []*sdkcompute.ResourceSKU, location string) map[string]*sdkcompute.ResourceSKU {
	vmskus := map[string]*sdkcompute.ResourceSKU{}
	for _, sku := range skus {
		// TODO(mjudeikis): At some point some SKU's stopped returning zones and
		// locations. IcM is open with MSFT but this might take a while.
		// Revert once we find out right behaviour.
		// https://github.com/Azure/ARO-RP/issues/1515
		if len(sku.Locations) == 0 || !strings.EqualFold(*(sku.Locations)[0], location) ||
			*sku.ResourceType != "virtualMachines" {
			continue
		}

		if len(sku.LocationInfo) == 0 { // happened in eastus2euap
			continue
		}

		// We copy only part of the object so we don't have to keep
		// a lot of data in memory.
		vmskus[*sku.Name] = &sdkcompute.ResourceSKU{
			Name:         sku.Name,
			Restrictions: sku.Restrictions,
			LocationInfo: sku.LocationInfo,
			Capabilities: sku.Capabilities,
		}
	}

	return vmskus
}

// SupportedOSDisk returns the type of OSDisk for the given resource. Most VMs will use Premium disks but some SKUs only support Standard SSDs
func SupportedOSDisk(vmSku *sdkcompute.ResourceSKU) string {
	if HasCapability(vmSku, premiumDiskCapability) {
		return premiumDisk
	}
	return standardDisk
}

func GetVMSkusForCurrentRegion(ctx context.Context, resourceSkusClient armcompute.ResourceSKUsClient, location string) (map[string]*sdkcompute.ResourceSKU, error) {
	filter := fmt.Sprintf("location eq %s", location)
	skus, err := resourceSkusClient.List(ctx, filter, false)
	if err != nil {
		return nil, errors.Join(ErrListVMResourceSKUs, err)
	}

	return FilterVMSizes(skus, location), nil
}
