package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
)

// Zones returns zone information for the resource SKU
func Zones(sku *mgmtcompute.ResourceSku) []string {
	if sku.LocationInfo == nil ||
		len(*sku.LocationInfo) == 0 ||
		(*sku.LocationInfo)[0].Zones == nil {
		return nil
	}

	return *(*sku.LocationInfo)[0].Zones
}

// HasCapability checks whether given resource SKU has specific capability
func HasCapability(sku *mgmtcompute.ResourceSku, capabilityName string) bool {
	if sku.Capabilities == nil {
		return false
	}

	for _, c := range *sku.Capabilities {
		if *c.Name == capabilityName {
			return *c.Value == "True"
		}
	}

	return false
}

// IsRestricted checks whether given resource SKU is restricted in a given location
func IsRestricted(skus map[string]*mgmtcompute.ResourceSku, location, VMSize string) bool {
	for _, restriction := range *skus[VMSize].Restrictions {
		for _, restrictedLocation := range *restriction.RestrictionInfo.Locations {
			if restrictedLocation == location {
				return true
			}
		}
	}

	return false
}

// FilterVMSizes filters resource SKU by location and returns only virtual machines, their names, restrictions, location info, and capabilities.
func FilterVMSizes(skus []mgmtcompute.ResourceSku, location string) map[string]*mgmtcompute.ResourceSku {
	vmskus := map[string]*mgmtcompute.ResourceSku{}
	for _, sku := range skus {
		// TODO(mjudeikis): At some point some SKU's stopped returning zones and
		// locations. IcM is open with MSFT but this might take a while.
		// Revert once we find out right behaviour.
		// https://github.com/Azure/ARO-RP/issues/1515
		if len(*sku.Locations) == 0 || !strings.EqualFold((*sku.Locations)[0], location) ||
			*sku.ResourceType != "virtualMachines" {
			continue
		}

		if len(*sku.LocationInfo) == 0 { // happened in eastus2euap
			continue
		}

		// We copy only part of the object so we don't have to keep
		// a lot of data in memory.
		vmskus[*sku.Name] = &mgmtcompute.ResourceSku{
			Name:         sku.Name,
			Restrictions: sku.Restrictions,
			LocationInfo: sku.LocationInfo,
			Capabilities: sku.Capabilities,
		}
	}

	return vmskus
}
