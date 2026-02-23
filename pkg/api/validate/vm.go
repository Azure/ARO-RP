package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
)

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
	supportedVMSizesByRole := getSupportedVMSizesByRole(isCI)

	if isMaster {
		_, supportedAsMaster := supportedVMSizesByRole[vms.VMRoleMaster][vmSize]
		return supportedAsMaster
	}

	_, supportedAsWorker := supportedVMSizesByRole[vms.VMRoleWorker][vmSize]
	return supportedAsWorker
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
	// Check version-specific restrictions
	if isMaster {
		if sizeInfo, exists := vms.SupportedMasterVMSizes[vmSize]; exists && sizeInfo.MinimumVersion != nil {
			return clusterVersion.Gt(sizeInfo.MinimumVersion) || clusterVersion.Eq(sizeInfo.MinimumVersion)
		}
	} else {
		if sizeInfo, exists := vms.SupportedWorkerVMSizes[vmSize]; exists && sizeInfo.MinimumVersion != nil {
			return clusterVersion.Gt(sizeInfo.MinimumVersion) || clusterVersion.Eq(sizeInfo.MinimumVersion)
		}
	}

	// VM size has no version restrictions or passed all checks
	return true
}

func VMSizeFromName(vmSize vms.VMSize) (vms.VMSizeStruct, bool) {
	return vms.LookupVMSize(vmSize)
}
