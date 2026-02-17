package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/version"
)

// Public facing document which lists supported VM Sizes:
// https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes

// To add new instance types, needs Project Management's involvement and instructions are below.,
// https://github.com/Azure/ARO-RP/blob/master/docs/adding-new-instance-types.md

const VMRoleMaster string = "master"
const VMRoleWorker string = "worker"

var supportedVMSizesByRoleMap = map[string]map[api.VMSize]api.VMSizeStruct{
	VMRoleMaster: supportedMasterVmSizes,
	VMRoleWorker: supportedWorkerVmSizes,
}

func SupportedVMSizesByRole(vmRole string) map[api.VMSize]api.VMSizeStruct {
	supportedvmsizes, exists := supportedVMSizesByRoleMap[vmRole]
	if !exists {
		return nil
	}
	return supportedvmsizes
}

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, isMaster bool) bool {
	if isMaster {
		_, supportedAsMaster := SupportedVMSizesByRole(VMRoleMaster)[vmSize]
		return supportedAsMaster
	}

	_, supportedAsWorker := SupportedVMSizesByRole(VMRoleWorker)[vmSize]
	return supportedAsWorker
}

// VMSizeIsValidForVersion validates VM size with version-specific restrictions
func VMSizeIsValidForVersion(vmSize api.VMSize, isMaster bool, v string) bool {
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
		if minVersion, exists := masterVmSizesWithMinimumVersion[vmSize]; exists {
			return clusterVersion.Gt(minVersion) || clusterVersion.Eq(minVersion)
		}
	} else {
		if minVersion, exists := workerVmSizesWithMinimumVersion[vmSize]; exists {
			return clusterVersion.Gt(minVersion) || clusterVersion.Eq(minVersion)
		}
	}

	// VM size has no version restrictions or passed all checks
	return true
}

func VMSizeFromName(vmSize api.VMSize) (api.VMSizeStruct, bool) {
	//this is for development purposes only
	switch vmSize {
	case api.VMSizeStandardD2sV3:
		return api.VMSizeStandardD2sV3Struct, true
	case api.VMSizeStandardD2sV4:
		return api.VMSizeStandardD2sV4Struct, true
	case api.VMSizeStandardD2sV5:
		return api.VMSizeStandardD2sV5Struct, true
	}

	if size, ok := SupportedVMSizesByRole(VMRoleWorker)[vmSize]; ok {
		return size, true
	}

	if size, ok := SupportedVMSizesByRole(VMRoleMaster)[vmSize]; ok {
		return size, true
	}

	return api.VMSizeStruct{}, false
}
