package vms

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"maps"
	"sort"

	"github.com/Azure/ARO-RP/pkg/api"
)

var SupportedMasterVmSizes = map[api.VMSize]VMSizeStruct{
	// General purpose
	api.VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: VMSizeStandardD32sV3Struct,

	api.VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	api.VMSizeStandardD16sV4: VMSizeStandardD16sV4Struct,
	api.VMSizeStandardD32sV4: VMSizeStandardD32sV4Struct,

	api.VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	api.VMSizeStandardD16sV5: VMSizeStandardD16sV5Struct,
	api.VMSizeStandardD32sV5: VMSizeStandardD32sV5Struct,

	api.VMSizeStandardD8asV4:  VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD16asV4: VMSizeStandardD16asV4Struct,
	api.VMSizeStandardD32asV4: VMSizeStandardD32asV4Struct,

	api.VMSizeStandardD8asV5:  VMSizeStandardD8asV5Struct,
	api.VMSizeStandardD16asV5: VMSizeStandardD16asV5Struct,
	api.VMSizeStandardD32asV5: VMSizeStandardD32asV5Struct,

	api.VMSizeStandardD8dsV5:  VMSizeStandardD8dsV5Struct,
	api.VMSizeStandardD16dsV5: VMSizeStandardD16dsV5Struct,
	api.VMSizeStandardD32dsV5: VMSizeStandardD32dsV5Struct,

	// Memory optimized
	api.VMSizeStandardE8sV3:  VMSizeStandardE8sV3Struct,
	api.VMSizeStandardE16sV3: VMSizeStandardE16sV3Struct,
	api.VMSizeStandardE32sV3: VMSizeStandardE32sV3Struct,

	api.VMSizeStandardE8sV4:  VMSizeStandardE8sV4Struct,
	api.VMSizeStandardE16sV4: VMSizeStandardE16sV4Struct,
	api.VMSizeStandardE20sV4: VMSizeStandardE20sV4Struct,
	api.VMSizeStandardE32sV4: VMSizeStandardE32sV4Struct,
	api.VMSizeStandardE48sV4: VMSizeStandardE48sV4Struct,
	api.VMSizeStandardE64sV4: VMSizeStandardE64sV4Struct,

	api.VMSizeStandardE8sV5:  VMSizeStandardE8sV5Struct,
	api.VMSizeStandardE16sV5: VMSizeStandardE16sV5Struct,
	api.VMSizeStandardE20sV5: VMSizeStandardE20sV5Struct,
	api.VMSizeStandardE32sV5: VMSizeStandardE32sV5Struct,
	api.VMSizeStandardE48sV5: VMSizeStandardE48sV5Struct,
	api.VMSizeStandardE64sV5: VMSizeStandardE64sV5Struct,
	api.VMSizeStandardE96sV5: VMSizeStandardE96sV5Struct,

	api.VMSizeStandardE4asV4:  VMSizeStandardE4asV4Struct,
	api.VMSizeStandardE8asV4:  VMSizeStandardE8asV4Struct,
	api.VMSizeStandardE16asV4: VMSizeStandardE16asV4Struct,
	api.VMSizeStandardE20asV4: VMSizeStandardE20asV4Struct,
	api.VMSizeStandardE32asV4: VMSizeStandardE32asV4Struct,
	api.VMSizeStandardE48asV4: VMSizeStandardE48asV4Struct,
	api.VMSizeStandardE64asV4: VMSizeStandardE64asV4Struct,
	api.VMSizeStandardE96asV4: VMSizeStandardE96asV4Struct,

	api.VMSizeStandardE8asV5:  VMSizeStandardE8asV5Struct,
	api.VMSizeStandardE16asV5: VMSizeStandardE16asV5Struct,
	api.VMSizeStandardE20asV5: VMSizeStandardE20asV5Struct,
	api.VMSizeStandardE32asV5: VMSizeStandardE32asV5Struct,
	api.VMSizeStandardE48asV5: VMSizeStandardE48asV5Struct,
	api.VMSizeStandardE64asV5: VMSizeStandardE64asV5Struct,
	api.VMSizeStandardE96asV5: VMSizeStandardE96asV5Struct,

	api.VMSizeStandardE64isV3:   VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE80isV4:   VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104isV5:  VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idsV5: VMSizeStandardE104idsV5Struct,

	// Compute optimized
	api.VMSizeStandardF72sV2: VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: VMSizeStandardM128msStruct,

	api.VMSizeStandardD4sV6:  VMSizeStandardD4sV6Struct,
	api.VMSizeStandardD8sV6:  VMSizeStandardD8sV6Struct,
	api.VMSizeStandardD16sV6: VMSizeStandardD16sV6Struct,
	api.VMSizeStandardD32sV6: VMSizeStandardD32sV6Struct,
	api.VMSizeStandardD48sV6: VMSizeStandardD48sV6Struct,
	api.VMSizeStandardD64sV6: VMSizeStandardD64sV6Struct,
	api.VMSizeStandardD96sV6: VMSizeStandardD96sV6Struct,

	api.VMSizeStandardD4dsV6:  VMSizeStandardD4dsV6Struct,
	api.VMSizeStandardD8dsV6:  VMSizeStandardD8dsV6Struct,
	api.VMSizeStandardD16dsV6: VMSizeStandardD16dsV6Struct,
	api.VMSizeStandardD32dsV6: VMSizeStandardD32dsV6Struct,
	api.VMSizeStandardD48dsV6: VMSizeStandardD48dsV6Struct,
	api.VMSizeStandardD64dsV6: VMSizeStandardD64dsV6Struct,
	api.VMSizeStandardD96dsV6: VMSizeStandardD96dsV6Struct,
}

var SupportedWorkerVmSizes = map[api.VMSize]VMSizeStruct{
	// General purpose
	api.VMSizeStandardD4sV3:  VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: VMSizeStandardD32sV3Struct,

	api.VMSizeStandardD4sV4:  VMSizeStandardD4sV4Struct,
	api.VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	api.VMSizeStandardD16sV4: VMSizeStandardD16sV4Struct,
	api.VMSizeStandardD32sV4: VMSizeStandardD32sV4Struct,
	api.VMSizeStandardD64sV4: VMSizeStandardD64sV4Struct,

	api.VMSizeStandardD4sV5:  VMSizeStandardD4sV5Struct,
	api.VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	api.VMSizeStandardD16sV5: VMSizeStandardD16sV5Struct,
	api.VMSizeStandardD32sV5: VMSizeStandardD32sV5Struct,
	api.VMSizeStandardD64sV5: VMSizeStandardD64sV5Struct,
	api.VMSizeStandardD96sV5: VMSizeStandardD96sV5Struct,

	api.VMSizeStandardD4asV4:  VMSizeStandardD4asV4Struct,
	api.VMSizeStandardD8asV4:  VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD16asV4: VMSizeStandardD16asV4Struct,
	api.VMSizeStandardD32asV4: VMSizeStandardD32asV4Struct,
	api.VMSizeStandardD64asV4: VMSizeStandardD64asV4Struct,
	api.VMSizeStandardD96asV4: VMSizeStandardD96asV4Struct,

	api.VMSizeStandardD4asV5:  VMSizeStandardD4asV5Struct,
	api.VMSizeStandardD8asV5:  VMSizeStandardD8asV5Struct,
	api.VMSizeStandardD16asV5: VMSizeStandardD16asV5Struct,
	api.VMSizeStandardD32asV5: VMSizeStandardD32asV5Struct,
	api.VMSizeStandardD64asV5: VMSizeStandardD64asV5Struct,
	api.VMSizeStandardD96asV5: VMSizeStandardD96asV5Struct,

	api.VMSizeStandardD4dsV5:  VMSizeStandardD4dsV5Struct,
	api.VMSizeStandardD8dsV5:  VMSizeStandardD8dsV5Struct,
	api.VMSizeStandardD16dsV5: VMSizeStandardD16dsV5Struct,
	api.VMSizeStandardD32dsV5: VMSizeStandardD32dsV5Struct,
	api.VMSizeStandardD64dsV5: VMSizeStandardD64dsV5Struct,
	api.VMSizeStandardD96dsV5: VMSizeStandardD96dsV5Struct,

	// Memory optimized
	api.VMSizeStandardE4sV3:  VMSizeStandardE4sV3Struct,
	api.VMSizeStandardE8sV3:  VMSizeStandardE8sV3Struct,
	api.VMSizeStandardE16sV3: VMSizeStandardE16sV3Struct,
	api.VMSizeStandardE32sV3: VMSizeStandardE32sV3Struct,

	api.VMSizeStandardE2sV4:  VMSizeStandardE2sV4Struct,
	api.VMSizeStandardE4sV4:  VMSizeStandardE4sV4Struct,
	api.VMSizeStandardE8sV4:  VMSizeStandardE8sV4Struct,
	api.VMSizeStandardE16sV4: VMSizeStandardE16sV4Struct,
	api.VMSizeStandardE20sV4: VMSizeStandardE20sV4Struct,
	api.VMSizeStandardE32sV4: VMSizeStandardE32sV4Struct,
	api.VMSizeStandardE48sV4: VMSizeStandardE48sV4Struct,
	api.VMSizeStandardE64sV4: VMSizeStandardE64sV4Struct,

	api.VMSizeStandardE2sV5:  VMSizeStandardE2sV5Struct,
	api.VMSizeStandardE4sV5:  VMSizeStandardE4sV5Struct,
	api.VMSizeStandardE8sV5:  VMSizeStandardE8sV5Struct,
	api.VMSizeStandardE16sV5: VMSizeStandardE16sV5Struct,
	api.VMSizeStandardE20sV5: VMSizeStandardE20sV5Struct,
	api.VMSizeStandardE32sV5: VMSizeStandardE32sV5Struct,
	api.VMSizeStandardE48sV5: VMSizeStandardE48sV5Struct,
	api.VMSizeStandardE64sV5: VMSizeStandardE64sV5Struct,
	api.VMSizeStandardE96sV5: VMSizeStandardE96sV5Struct,

	api.VMSizeStandardE4asV4:  VMSizeStandardE4asV4Struct,
	api.VMSizeStandardE8asV4:  VMSizeStandardE8asV4Struct,
	api.VMSizeStandardE16asV4: VMSizeStandardE16asV4Struct,
	api.VMSizeStandardE20asV4: VMSizeStandardE20asV4Struct,
	api.VMSizeStandardE32asV4: VMSizeStandardE32asV4Struct,
	api.VMSizeStandardE48asV4: VMSizeStandardE48asV4Struct,
	api.VMSizeStandardE64asV4: VMSizeStandardE64asV4Struct,
	api.VMSizeStandardE96asV4: VMSizeStandardE96asV4Struct,

	api.VMSizeStandardE8asV5:  VMSizeStandardE8asV5Struct,
	api.VMSizeStandardE16asV5: VMSizeStandardE16asV5Struct,
	api.VMSizeStandardE20asV5: VMSizeStandardE20asV5Struct,
	api.VMSizeStandardE32asV5: VMSizeStandardE32asV5Struct,
	api.VMSizeStandardE48asV5: VMSizeStandardE48asV5Struct,
	api.VMSizeStandardE64asV5: VMSizeStandardE64asV5Struct,
	api.VMSizeStandardE96asV5: VMSizeStandardE96asV5Struct,

	api.VMSizeStandardE64isV3:   VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE80isV4:   VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104isV5:  VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idsV5: VMSizeStandardE104idsV5Struct,

	// Compute optimized
	api.VMSizeStandardF4sV2:  VMSizeStandardF4sV2Struct,
	api.VMSizeStandardF8sV2:  VMSizeStandardF8sV2Struct,
	api.VMSizeStandardF16sV2: VMSizeStandardF16sV2Struct,
	api.VMSizeStandardF32sV2: VMSizeStandardF32sV2Struct,
	api.VMSizeStandardF72sV2: VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: VMSizeStandardM128msStruct,

	// Storage optimized
	api.VMSizeStandardL4s:  VMSizeStandardL4sStruct,
	api.VMSizeStandardL8s:  VMSizeStandardL8sStruct,
	api.VMSizeStandardL16s: VMSizeStandardL16sStruct,
	api.VMSizeStandardL32s: VMSizeStandardL32sStruct,

	api.VMSizeStandardL8sV2:  VMSizeStandardL8sV2Struct,
	api.VMSizeStandardL16sV2: VMSizeStandardL16sV2Struct,
	api.VMSizeStandardL32sV2: VMSizeStandardL32sV2Struct,
	api.VMSizeStandardL48sV2: VMSizeStandardL48sV2Struct,
	api.VMSizeStandardL64sV2: VMSizeStandardL64sV2Struct,

	api.VMSizeStandardL8sV3:  VMSizeStandardL8sV3Struct,
	api.VMSizeStandardL16sV3: VMSizeStandardL16sV3Struct,
	api.VMSizeStandardL32sV3: VMSizeStandardL32sV3Struct,
	api.VMSizeStandardL48sV3: VMSizeStandardL48sV3Struct,
	api.VMSizeStandardL64sV3: VMSizeStandardL64sV3Struct,

	api.VMSizeStandardL4sV4:  VMSizeStandardL4sV4Struct,
	api.VMSizeStandardL8sV4:  VMSizeStandardL8sV4Struct,
	api.VMSizeStandardL16sV4: VMSizeStandardL16sV4Struct,
	api.VMSizeStandardL32sV4: VMSizeStandardL32sV4Struct,
	api.VMSizeStandardL48sV4: VMSizeStandardL48sV4Struct,
	api.VMSizeStandardL64sV4: VMSizeStandardL64sV4Struct,
	api.VMSizeStandardL80sV4: VMSizeStandardL80sV4Struct,

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	api.VMSizeStandardNC4asT4V3:  VMSizeStandardNC4asT4V3Struct,
	api.VMSizeStandardNC8asT4V3:  VMSizeStandardNC8asT4V3Struct,
	api.VMSizeStandardNC16asT4V3: VMSizeStandardNC16asT4V3Struct,
	api.VMSizeStandardNC64asT4V3: VMSizeStandardNC64asT4V3Struct,

	api.VMSizeStandardNC6sV3:   VMSizeStandardNC6sV3Struct,
	api.VMSizeStandardNC12sV3:  VMSizeStandardNC12sV3Struct,
	api.VMSizeStandardNC24sV3:  VMSizeStandardNC24sV3Struct,
	api.VMSizeStandardNC24rsV3: VMSizeStandardNC24rsV3Struct,

	api.VMSizeStandardD4sV6:  VMSizeStandardD4sV6Struct,
	api.VMSizeStandardD8sV6:  VMSizeStandardD8sV6Struct,
	api.VMSizeStandardD16sV6: VMSizeStandardD16sV6Struct,
	api.VMSizeStandardD32sV6: VMSizeStandardD32sV6Struct,
	api.VMSizeStandardD48sV6: VMSizeStandardD48sV6Struct,
	api.VMSizeStandardD64sV6: VMSizeStandardD64sV6Struct,
	api.VMSizeStandardD96sV6: VMSizeStandardD96sV6Struct,

	api.VMSizeStandardD4dsV6:  VMSizeStandardD4dsV6Struct,
	api.VMSizeStandardD8dsV6:  VMSizeStandardD8dsV6Struct,
	api.VMSizeStandardD16dsV6: VMSizeStandardD16dsV6Struct,
	api.VMSizeStandardD32dsV6: VMSizeStandardD32dsV6Struct,
	api.VMSizeStandardD48dsV6: VMSizeStandardD48dsV6Struct,
	api.VMSizeStandardD64dsV6: VMSizeStandardD64dsV6Struct,
	api.VMSizeStandardD96dsV6: VMSizeStandardD96dsV6Struct,

	api.VMSizeStandardD4lsV6:  VMSizeStandardD4lsV6Struct,
	api.VMSizeStandardD8lsV6:  VMSizeStandardD8lsV6Struct,
	api.VMSizeStandardD16lsV6: VMSizeStandardD16lsV6Struct,
	api.VMSizeStandardD32lsV6: VMSizeStandardD32lsV6Struct,
	api.VMSizeStandardD48lsV6: VMSizeStandardD48lsV6Struct,
	api.VMSizeStandardD64lsV6: VMSizeStandardD64lsV6Struct,
	api.VMSizeStandardD96lsV6: VMSizeStandardD96lsV6Struct,

	api.VMSizeStandardD4ldsV6:  VMSizeStandardD4ldsV6Struct,
	api.VMSizeStandardD8ldsV6:  VMSizeStandardD8ldsV6Struct,
	api.VMSizeStandardD16ldsV6: VMSizeStandardD16ldsV6Struct,
	api.VMSizeStandardD32ldsV6: VMSizeStandardD32ldsV6Struct,
	api.VMSizeStandardD48ldsV6: VMSizeStandardD48ldsV6Struct,
	api.VMSizeStandardD64ldsV6: VMSizeStandardD64ldsV6Struct,
	api.VMSizeStandardD96ldsV6: VMSizeStandardD96ldsV6Struct,
}

var SupportedMasterVmSizesForInternalUser = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD4sV3: VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD4sV4: VMSizeStandardD4sV4Struct,
	api.VMSizeStandardD4sV5: VMSizeStandardD4sV5Struct,
	api.VMSizeStandardD4sV6: VMSizeStandardD4sV6Struct,
}

var SupportedWorkerVmSizesForInternalUser = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD2sV3: VMSizeStandardD2sV3Struct,
	api.VMSizeStandardD2sV4: VMSizeStandardD2sV4Struct,
	api.VMSizeStandardD2sV5: VMSizeStandardD2sV5Struct,
	api.VMSizeStandardD2sV6: VMSizeStandardD2sV6Struct,
}

func init() {
	maps.Copy(SupportedMasterVmSizesForInternalUser, SupportedMasterVmSizes)
	maps.Copy(SupportedWorkerVmSizesForInternalUser, SupportedWorkerVmSizes)
}

// MinMasterVMSizes contains the smallest supported master VM size for each
// general-purpose D-series family. Used by test/CI/dev tooling to select
// cost-effective sizes while spreading quota across families.
var MinMasterVMSizes = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	api.VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	api.VMSizeStandardD8asV4: VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD8asV5: VMSizeStandardD8asV5Struct,
	api.VMSizeStandardD8dsV5: VMSizeStandardD8dsV5Struct,
}

// MinWorkerVMSizes contains the smallest supported worker VM size for each
// general-purpose D-series family.
var MinWorkerVMSizes = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD2sV3:  VMSizeStandardD2sV3Struct,
	api.VMSizeStandardD2sV4:  VMSizeStandardD2sV4Struct,
	api.VMSizeStandardD2sV5:  VMSizeStandardD2sV5Struct,
	api.VMSizeStandardD4sV3:  VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD4sV4:  VMSizeStandardD4sV4Struct,
	api.VMSizeStandardD4sV5:  VMSizeStandardD4sV5Struct,
	api.VMSizeStandardD4asV4: VMSizeStandardD4asV4Struct,
	api.VMSizeStandardD4asV5: VMSizeStandardD4asV5Struct,
	api.VMSizeStandardD4dsV5: VMSizeStandardD4dsV5Struct,
}

const (
	VMRoleMaster = "master"
	VMRoleWorker = "worker"
)

// MinVMSizesForRole returns the minimum VM sizes for a role, sorted by core
// count (smallest first). This is used by test/CI/dev tooling.
func MinVMSizesForRole(vmRole string) []api.VMSize {
	var m map[api.VMSize]VMSizeStruct
	switch vmRole {
	case VMRoleMaster:
		m = MinMasterVMSizes
	case VMRoleWorker:
		m = MinWorkerVMSizes
	default:
		return nil
	}

	type entry struct {
		size      api.VMSize
		coreCount int
	}

	entries := make([]entry, 0, len(m))
	for sz, info := range m {
		entries = append(entries, entry{size: sz, coreCount: info.CoreCount})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].coreCount != entries[j].coreCount {
			return entries[i].coreCount < entries[j].coreCount
		}
		return entries[i].size < entries[j].size
	})

	result := make([]api.VMSize, len(entries))
	for i, e := range entries {
		result[i] = e.size
	}
	return result
}
