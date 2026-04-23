package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
)

// Public facing document which lists supported VM Sizes:
// https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes

// To add new instance types, needs Project Management's involvement and instructions are below.,
// https://github.com/Azure/ARO-RP/blob/master/docs/adding-new-instance-types.md

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func getSupportedVMSizesByRole(isCI bool) map[vms.VMRole]map[vms.VMSize]vms.VMSizeStruct {
	if isCI {
		return vms.SupportedVMSizesByRoleForTesting
	}
	return vms.SupportedVMSizesByRole
}

func VMSizeIsValid(vmSize vms.VMSize, isMaster bool, isCI bool) bool {
	role := vms.VMRoleWorker
	if isMaster {
		role = vms.VMRoleMaster
	}

	supportedSizes := getSupportedVMSizesByRole(isCI)
	_, supported := supportedSizes[role][vmSize]
	return supported
}

// VMSizeIsValidForVersion validates VM size with version-specific restrictions
func VMSizeIsValidForVersion(vmSize vms.VMSize, isMaster bool, v string, isCI bool) bool {
	// First check basic validity
	if !VMSizeIsValid(vmSize, isMaster, isCI) {
		return false
	}

	clusterVersion, err := version.ParseVersion(v)
	if err != nil {
		return false
	}

	role := vms.VMRoleWorker
	if isMaster {
		role = vms.VMRoleMaster
	}

	supportedSizes := getSupportedVMSizesByRole(isCI)
	sizeInfo := supportedSizes[role][vmSize]

	// If the VM size has a minimum version requirement, check it
	if sizeInfo.MinimumVersion != nil {
		return clusterVersion.Gt(sizeInfo.MinimumVersion) || clusterVersion.Eq(sizeInfo.MinimumVersion)
	}

	// VM size has no version restrictions
	return true
}

func VMSizeFromName(vmSize vms.VMSize) (vms.VMSizeStruct, bool) {
	return vms.LookupVMSize(vmSize)
}
