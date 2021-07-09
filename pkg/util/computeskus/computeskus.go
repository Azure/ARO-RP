package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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

// Capabilities checks whether given resource SKU has specific capability
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
