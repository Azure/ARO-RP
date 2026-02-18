package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
)

var supportedVMSizesByRoleMap = map[vms.VMRole]map[vms.VMSize]vms.VMSizeStruct{
	vms.VMRoleMaster: vms.SupportedMasterVMSizes,
	vms.VMRoleWorker: vms.SupportedWorkerVMSizes,
}

func SupportedVMSizesByRole(vmRole vms.VMRole) map[vms.VMSize]vms.VMSizeStruct {
	supportedvmsizes, exists := supportedVMSizesByRoleMap[vmRole]
	if !exists {
		return nil
	}
	return supportedvmsizes
}

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize vms.VMSize, isMaster bool) bool {
	if isMaster {
		_, supportedAsMaster := SupportedVMSizesByRole(vms.VMRoleMaster)[vmSize]
		return supportedAsMaster
	}

	_, supportedAsWorker := SupportedVMSizesByRole(vms.VMRoleWorker)[vmSize]
	return supportedAsWorker
}

// VMSizeIsValidForVersion validates VM size with version-specific restrictions
func VMSizeIsValidForVersion(vmSize vms.VMSize, isMaster bool, v string) bool {
	// First check basic validity
	if !VMSizeIsValid(vmSize, isMaster) {
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
	// this is for development purposes only
	switch vmSize {
	case vms.VMSizeStandardD2sV3:
		return vms.VMSizeStandardD2sV3Struct, true
	case vms.VMSizeStandardD2sV4:
		return vms.VMSizeStandardD2sV4Struct, true
	case vms.VMSizeStandardD2sV5:
		return vms.VMSizeStandardD2sV5Struct, true
	}

	if size, ok := SupportedVMSizesByRole(vms.VMRoleWorker)[vmSize]; ok {
		return size, true
	}

	if size, ok := SupportedVMSizesByRole(vms.VMRoleMaster)[vmSize]; ok {
		return size, true
	}

	return vms.VMSizeStruct{}, false
}
