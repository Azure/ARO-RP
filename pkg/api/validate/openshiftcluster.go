package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

// TODO: remove the VM name validation after https://bugzilla.redhat.com/show_bug.cgi?id=2093044 is resolved

// Max length of a ARO cluster name is determined by the length of the generated availability set
// in the machine-api provider.  It varies per region, but to remain consistent across non-zonal regions
// we hardcode this to 19, which is the max for all non-zonal regions.
//
// the generated availability set must be <= 80 characters and can be calculated below
// <cluster-name>-XXXXX_<cluster-name>-XXXXX-worker-<region>-as
//   XXXXX:          represents the 5 digit cluster infraID
//   <cluster-name>: the name of the cluster
//   <region>:       the short-name of the region
const MaxClusterNameLength = 19

// nonZonalRegions are defined by the Compute List SKUs API not returning zones within the VM objects
//
// This can be queried for a location by running `az vm list-skus -l <region> --all --zone`
// If the object is empty that means the location does not support VMs deployed into
// availability zones
var nonZonalRegions = map[string]bool{
	"australiacentral":   true,
	"australiacentral2":  true,
	"australiasoutheast": true,
	"brazilsoutheast":    true,
	"canadaeast":         true,
	"japanwest":          true,
	"northcentralus":     true,
	"norwaywest":         true,
	"southindia":         true,
	"switzerlandwest":    true,
	"uaenorth":           true,
	"ukwest":             true,
	"westcentralus":      true,
	"westus":             true,
}

// OpenShiftClusterNameLength validates that the name does not exceed the maximumLength
// which is in place for non-zonal regions due to https://bugzilla.redhat.com/show_bug.cgi?id=2093044
func OpenShiftClusterNameLength(name, location string) bool {
	if nonZonalRegions[strings.ToLower(location)] && len(name) > MaxClusterNameLength {
		return false
	}

	return true
}
