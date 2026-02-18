package vms

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"maps"
	"math/rand/v2"
	"slices"
	"sort"
)

// Public facing document which lists supported VM Sizes:
// https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes

// To add new instance types, needs Project Management's involvement and instructions are below.,
// https://github.com/Azure/ARO-RP/blob/master/docs/adding-new-instance-types.md

var SupportedMasterVmSizes = map[VMSize]VMSizeStruct{
	// General purpose
	VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	VMSizeStandardD16sV3: VMSizeStandardD16sV3Struct,
	VMSizeStandardD32sV3: VMSizeStandardD32sV3Struct,

	VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	VMSizeStandardD16sV4: VMSizeStandardD16sV4Struct,
	VMSizeStandardD32sV4: VMSizeStandardD32sV4Struct,

	VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	VMSizeStandardD16sV5: VMSizeStandardD16sV5Struct,
	VMSizeStandardD32sV5: VMSizeStandardD32sV5Struct,

	VMSizeStandardD8asV4:  VMSizeStandardD8asV4Struct,
	VMSizeStandardD16asV4: VMSizeStandardD16asV4Struct,
	VMSizeStandardD32asV4: VMSizeStandardD32asV4Struct,

	VMSizeStandardD8asV5:  VMSizeStandardD8asV5Struct,
	VMSizeStandardD16asV5: VMSizeStandardD16asV5Struct,
	VMSizeStandardD32asV5: VMSizeStandardD32asV5Struct,

	VMSizeStandardD8dsV5:  VMSizeStandardD8dsV5Struct,
	VMSizeStandardD16dsV5: VMSizeStandardD16dsV5Struct,
	VMSizeStandardD32dsV5: VMSizeStandardD32dsV5Struct,

	// Memory optimized
	VMSizeStandardE8sV3:  VMSizeStandardE8sV3Struct,
	VMSizeStandardE16sV3: VMSizeStandardE16sV3Struct,
	VMSizeStandardE32sV3: VMSizeStandardE32sV3Struct,

	VMSizeStandardE8sV4:  VMSizeStandardE8sV4Struct,
	VMSizeStandardE16sV4: VMSizeStandardE16sV4Struct,
	VMSizeStandardE20sV4: VMSizeStandardE20sV4Struct,
	VMSizeStandardE32sV4: VMSizeStandardE32sV4Struct,
	VMSizeStandardE48sV4: VMSizeStandardE48sV4Struct,
	VMSizeStandardE64sV4: VMSizeStandardE64sV4Struct,

	VMSizeStandardE8sV5:  VMSizeStandardE8sV5Struct,
	VMSizeStandardE16sV5: VMSizeStandardE16sV5Struct,
	VMSizeStandardE20sV5: VMSizeStandardE20sV5Struct,
	VMSizeStandardE32sV5: VMSizeStandardE32sV5Struct,
	VMSizeStandardE48sV5: VMSizeStandardE48sV5Struct,
	VMSizeStandardE64sV5: VMSizeStandardE64sV5Struct,
	VMSizeStandardE96sV5: VMSizeStandardE96sV5Struct,

	VMSizeStandardE4asV4:  VMSizeStandardE4asV4Struct,
	VMSizeStandardE8asV4:  VMSizeStandardE8asV4Struct,
	VMSizeStandardE16asV4: VMSizeStandardE16asV4Struct,
	VMSizeStandardE20asV4: VMSizeStandardE20asV4Struct,
	VMSizeStandardE32asV4: VMSizeStandardE32asV4Struct,
	VMSizeStandardE48asV4: VMSizeStandardE48asV4Struct,
	VMSizeStandardE64asV4: VMSizeStandardE64asV4Struct,
	VMSizeStandardE96asV4: VMSizeStandardE96asV4Struct,

	VMSizeStandardE8asV5:  VMSizeStandardE8asV5Struct,
	VMSizeStandardE16asV5: VMSizeStandardE16asV5Struct,
	VMSizeStandardE20asV5: VMSizeStandardE20asV5Struct,
	VMSizeStandardE32asV5: VMSizeStandardE32asV5Struct,
	VMSizeStandardE48asV5: VMSizeStandardE48asV5Struct,
	VMSizeStandardE64asV5: VMSizeStandardE64asV5Struct,
	VMSizeStandardE96asV5: VMSizeStandardE96asV5Struct,

	VMSizeStandardE64isV3:   VMSizeStandardE64isV3Struct,
	VMSizeStandardE80isV4:   VMSizeStandardE80isV4Struct,
	VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4Struct,
	VMSizeStandardE104isV5:  VMSizeStandardE104isV5Struct,
	VMSizeStandardE104idsV5: VMSizeStandardE104idsV5Struct,

	// Compute optimized
	VMSizeStandardF72sV2: VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	VMSizeStandardM128ms: VMSizeStandardM128msStruct,

	VMSizeStandardD4sV6:  VMSizeStandardD4sV6Struct,
	VMSizeStandardD8sV6:  VMSizeStandardD8sV6Struct,
	VMSizeStandardD16sV6: VMSizeStandardD16sV6Struct,
	VMSizeStandardD32sV6: VMSizeStandardD32sV6Struct,
	VMSizeStandardD48sV6: VMSizeStandardD48sV6Struct,
	VMSizeStandardD64sV6: VMSizeStandardD64sV6Struct,
	VMSizeStandardD96sV6: VMSizeStandardD96sV6Struct,

	VMSizeStandardD4dsV6:  VMSizeStandardD4dsV6Struct,
	VMSizeStandardD8dsV6:  VMSizeStandardD8dsV6Struct,
	VMSizeStandardD16dsV6: VMSizeStandardD16dsV6Struct,
	VMSizeStandardD32dsV6: VMSizeStandardD32dsV6Struct,
	VMSizeStandardD48dsV6: VMSizeStandardD48dsV6Struct,
	VMSizeStandardD64dsV6: VMSizeStandardD64dsV6Struct,
	VMSizeStandardD96dsV6: VMSizeStandardD96dsV6Struct,
}

var SupportedWorkerVmSizes = map[VMSize]VMSizeStruct{
	// General purpose
	VMSizeStandardD4sV3:  VMSizeStandardD4sV3Struct,
	VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	VMSizeStandardD16sV3: VMSizeStandardD16sV3Struct,
	VMSizeStandardD32sV3: VMSizeStandardD32sV3Struct,

	VMSizeStandardD4sV4:  VMSizeStandardD4sV4Struct,
	VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	VMSizeStandardD16sV4: VMSizeStandardD16sV4Struct,
	VMSizeStandardD32sV4: VMSizeStandardD32sV4Struct,
	VMSizeStandardD64sV4: VMSizeStandardD64sV4Struct,

	VMSizeStandardD4sV5:  VMSizeStandardD4sV5Struct,
	VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	VMSizeStandardD16sV5: VMSizeStandardD16sV5Struct,
	VMSizeStandardD32sV5: VMSizeStandardD32sV5Struct,
	VMSizeStandardD64sV5: VMSizeStandardD64sV5Struct,
	VMSizeStandardD96sV5: VMSizeStandardD96sV5Struct,

	VMSizeStandardD4asV4:  VMSizeStandardD4asV4Struct,
	VMSizeStandardD8asV4:  VMSizeStandardD8asV4Struct,
	VMSizeStandardD16asV4: VMSizeStandardD16asV4Struct,
	VMSizeStandardD32asV4: VMSizeStandardD32asV4Struct,
	VMSizeStandardD64asV4: VMSizeStandardD64asV4Struct,
	VMSizeStandardD96asV4: VMSizeStandardD96asV4Struct,

	VMSizeStandardD4asV5:  VMSizeStandardD4asV5Struct,
	VMSizeStandardD8asV5:  VMSizeStandardD8asV5Struct,
	VMSizeStandardD16asV5: VMSizeStandardD16asV5Struct,
	VMSizeStandardD32asV5: VMSizeStandardD32asV5Struct,
	VMSizeStandardD64asV5: VMSizeStandardD64asV5Struct,
	VMSizeStandardD96asV5: VMSizeStandardD96asV5Struct,

	VMSizeStandardD4dsV5:  VMSizeStandardD4dsV5Struct,
	VMSizeStandardD8dsV5:  VMSizeStandardD8dsV5Struct,
	VMSizeStandardD16dsV5: VMSizeStandardD16dsV5Struct,
	VMSizeStandardD32dsV5: VMSizeStandardD32dsV5Struct,
	VMSizeStandardD64dsV5: VMSizeStandardD64dsV5Struct,
	VMSizeStandardD96dsV5: VMSizeStandardD96dsV5Struct,

	// Memory optimized
	VMSizeStandardE4sV3:  VMSizeStandardE4sV3Struct,
	VMSizeStandardE8sV3:  VMSizeStandardE8sV3Struct,
	VMSizeStandardE16sV3: VMSizeStandardE16sV3Struct,
	VMSizeStandardE32sV3: VMSizeStandardE32sV3Struct,

	VMSizeStandardE2sV4:  VMSizeStandardE2sV4Struct,
	VMSizeStandardE4sV4:  VMSizeStandardE4sV4Struct,
	VMSizeStandardE8sV4:  VMSizeStandardE8sV4Struct,
	VMSizeStandardE16sV4: VMSizeStandardE16sV4Struct,
	VMSizeStandardE20sV4: VMSizeStandardE20sV4Struct,
	VMSizeStandardE32sV4: VMSizeStandardE32sV4Struct,
	VMSizeStandardE48sV4: VMSizeStandardE48sV4Struct,
	VMSizeStandardE64sV4: VMSizeStandardE64sV4Struct,

	VMSizeStandardE2sV5:  VMSizeStandardE2sV5Struct,
	VMSizeStandardE4sV5:  VMSizeStandardE4sV5Struct,
	VMSizeStandardE8sV5:  VMSizeStandardE8sV5Struct,
	VMSizeStandardE16sV5: VMSizeStandardE16sV5Struct,
	VMSizeStandardE20sV5: VMSizeStandardE20sV5Struct,
	VMSizeStandardE32sV5: VMSizeStandardE32sV5Struct,
	VMSizeStandardE48sV5: VMSizeStandardE48sV5Struct,
	VMSizeStandardE64sV5: VMSizeStandardE64sV5Struct,
	VMSizeStandardE96sV5: VMSizeStandardE96sV5Struct,

	VMSizeStandardE4asV4:  VMSizeStandardE4asV4Struct,
	VMSizeStandardE8asV4:  VMSizeStandardE8asV4Struct,
	VMSizeStandardE16asV4: VMSizeStandardE16asV4Struct,
	VMSizeStandardE20asV4: VMSizeStandardE20asV4Struct,
	VMSizeStandardE32asV4: VMSizeStandardE32asV4Struct,
	VMSizeStandardE48asV4: VMSizeStandardE48asV4Struct,
	VMSizeStandardE64asV4: VMSizeStandardE64asV4Struct,
	VMSizeStandardE96asV4: VMSizeStandardE96asV4Struct,

	VMSizeStandardE8asV5:  VMSizeStandardE8asV5Struct,
	VMSizeStandardE16asV5: VMSizeStandardE16asV5Struct,
	VMSizeStandardE20asV5: VMSizeStandardE20asV5Struct,
	VMSizeStandardE32asV5: VMSizeStandardE32asV5Struct,
	VMSizeStandardE48asV5: VMSizeStandardE48asV5Struct,
	VMSizeStandardE64asV5: VMSizeStandardE64asV5Struct,
	VMSizeStandardE96asV5: VMSizeStandardE96asV5Struct,

	VMSizeStandardE64isV3:   VMSizeStandardE64isV3Struct,
	VMSizeStandardE80isV4:   VMSizeStandardE80isV4Struct,
	VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4Struct,
	VMSizeStandardE104isV5:  VMSizeStandardE104isV5Struct,
	VMSizeStandardE104idsV5: VMSizeStandardE104idsV5Struct,

	// Compute optimized
	VMSizeStandardF4sV2:  VMSizeStandardF4sV2Struct,
	VMSizeStandardF8sV2:  VMSizeStandardF8sV2Struct,
	VMSizeStandardF16sV2: VMSizeStandardF16sV2Struct,
	VMSizeStandardF32sV2: VMSizeStandardF32sV2Struct,
	VMSizeStandardF72sV2: VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	VMSizeStandardM128ms: VMSizeStandardM128msStruct,

	// Storage optimized
	VMSizeStandardL4s:  VMSizeStandardL4sStruct,
	VMSizeStandardL8s:  VMSizeStandardL8sStruct,
	VMSizeStandardL16s: VMSizeStandardL16sStruct,
	VMSizeStandardL32s: VMSizeStandardL32sStruct,

	VMSizeStandardL8sV2:  VMSizeStandardL8sV2Struct,
	VMSizeStandardL16sV2: VMSizeStandardL16sV2Struct,
	VMSizeStandardL32sV2: VMSizeStandardL32sV2Struct,
	VMSizeStandardL48sV2: VMSizeStandardL48sV2Struct,
	VMSizeStandardL64sV2: VMSizeStandardL64sV2Struct,

	VMSizeStandardL8sV3:  VMSizeStandardL8sV3Struct,
	VMSizeStandardL16sV3: VMSizeStandardL16sV3Struct,
	VMSizeStandardL32sV3: VMSizeStandardL32sV3Struct,
	VMSizeStandardL48sV3: VMSizeStandardL48sV3Struct,
	VMSizeStandardL64sV3: VMSizeStandardL64sV3Struct,

	VMSizeStandardL4sV4:  VMSizeStandardL4sV4Struct,
	VMSizeStandardL8sV4:  VMSizeStandardL8sV4Struct,
	VMSizeStandardL16sV4: VMSizeStandardL16sV4Struct,
	VMSizeStandardL32sV4: VMSizeStandardL32sV4Struct,
	VMSizeStandardL48sV4: VMSizeStandardL48sV4Struct,
	VMSizeStandardL64sV4: VMSizeStandardL64sV4Struct,
	VMSizeStandardL80sV4: VMSizeStandardL80sV4Struct,

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	VMSizeStandardNC4asT4V3:  VMSizeStandardNC4asT4V3Struct,
	VMSizeStandardNC8asT4V3:  VMSizeStandardNC8asT4V3Struct,
	VMSizeStandardNC16asT4V3: VMSizeStandardNC16asT4V3Struct,
	VMSizeStandardNC64asT4V3: VMSizeStandardNC64asT4V3Struct,

	VMSizeStandardNC6sV3:   VMSizeStandardNC6sV3Struct,
	VMSizeStandardNC12sV3:  VMSizeStandardNC12sV3Struct,
	VMSizeStandardNC24sV3:  VMSizeStandardNC24sV3Struct,
	VMSizeStandardNC24rsV3: VMSizeStandardNC24rsV3Struct,

	VMSizeStandardD4sV6:  VMSizeStandardD4sV6Struct,
	VMSizeStandardD8sV6:  VMSizeStandardD8sV6Struct,
	VMSizeStandardD16sV6: VMSizeStandardD16sV6Struct,
	VMSizeStandardD32sV6: VMSizeStandardD32sV6Struct,
	VMSizeStandardD48sV6: VMSizeStandardD48sV6Struct,
	VMSizeStandardD64sV6: VMSizeStandardD64sV6Struct,
	VMSizeStandardD96sV6: VMSizeStandardD96sV6Struct,

	VMSizeStandardD4dsV6:  VMSizeStandardD4dsV6Struct,
	VMSizeStandardD8dsV6:  VMSizeStandardD8dsV6Struct,
	VMSizeStandardD16dsV6: VMSizeStandardD16dsV6Struct,
	VMSizeStandardD32dsV6: VMSizeStandardD32dsV6Struct,
	VMSizeStandardD48dsV6: VMSizeStandardD48dsV6Struct,
	VMSizeStandardD64dsV6: VMSizeStandardD64dsV6Struct,
	VMSizeStandardD96dsV6: VMSizeStandardD96dsV6Struct,

	VMSizeStandardD4lsV6:  VMSizeStandardD4lsV6Struct,
	VMSizeStandardD8lsV6:  VMSizeStandardD8lsV6Struct,
	VMSizeStandardD16lsV6: VMSizeStandardD16lsV6Struct,
	VMSizeStandardD32lsV6: VMSizeStandardD32lsV6Struct,
	VMSizeStandardD48lsV6: VMSizeStandardD48lsV6Struct,
	VMSizeStandardD64lsV6: VMSizeStandardD64lsV6Struct,
	VMSizeStandardD96lsV6: VMSizeStandardD96lsV6Struct,

	VMSizeStandardD4ldsV6:  VMSizeStandardD4ldsV6Struct,
	VMSizeStandardD8ldsV6:  VMSizeStandardD8ldsV6Struct,
	VMSizeStandardD16ldsV6: VMSizeStandardD16ldsV6Struct,
	VMSizeStandardD32ldsV6: VMSizeStandardD32ldsV6Struct,
	VMSizeStandardD48ldsV6: VMSizeStandardD48ldsV6Struct,
	VMSizeStandardD64ldsV6: VMSizeStandardD64ldsV6Struct,
	VMSizeStandardD96ldsV6: VMSizeStandardD96ldsV6Struct,
}

var SupportedMasterVmSizesForInternalUser = map[VMSize]VMSizeStruct{
	VMSizeStandardD4sV3: VMSizeStandardD4sV3Struct,
	VMSizeStandardD4sV4: VMSizeStandardD4sV4Struct,
	VMSizeStandardD4sV5: VMSizeStandardD4sV5Struct,
	VMSizeStandardD4sV6: VMSizeStandardD4sV6Struct,
}

var SupportedWorkerVmSizesForInternalUser = map[VMSize]VMSizeStruct{
	VMSizeStandardD2sV3: VMSizeStandardD2sV3Struct,
	VMSizeStandardD2sV4: VMSizeStandardD2sV4Struct,
	VMSizeStandardD2sV5: VMSizeStandardD2sV5Struct,
	VMSizeStandardD2sV6: VMSizeStandardD2sV6Struct,
}

func init() {
	maps.Copy(SupportedMasterVmSizesForInternalUser, SupportedMasterVmSizes)
	maps.Copy(SupportedWorkerVmSizesForInternalUser, SupportedWorkerVmSizes)
}

// MinMasterVMSizes contains the smallest supported master VM size for each
// general-purpose D-series family. Used by test/CI/dev tooling to select
// cost-effective sizes while spreading quota across families.
var MinMasterVMSizes = map[VMSize]VMSizeStruct{
	VMSizeStandardD8sV3:  VMSizeStandardD8sV3Struct,
	VMSizeStandardD8sV4:  VMSizeStandardD8sV4Struct,
	VMSizeStandardD8sV5:  VMSizeStandardD8sV5Struct,
	VMSizeStandardD8asV4: VMSizeStandardD8asV4Struct,
	VMSizeStandardD8asV5: VMSizeStandardD8asV5Struct,
	VMSizeStandardD8dsV5: VMSizeStandardD8dsV5Struct,
}

// MinWorkerVMSizes contains the smallest supported worker VM size for each
// general-purpose D-series family.
var MinWorkerVMSizes = map[VMSize]VMSizeStruct{
	VMSizeStandardD2sV5:  VMSizeStandardD2sV5Struct,
	VMSizeStandardD4sV3:  VMSizeStandardD4sV3Struct,
	VMSizeStandardD4sV4:  VMSizeStandardD4sV4Struct,
	VMSizeStandardD4sV5:  VMSizeStandardD4sV5Struct,
	VMSizeStandardD4asV4: VMSizeStandardD4asV4Struct,
	VMSizeStandardD4asV5: VMSizeStandardD4asV5Struct,
	VMSizeStandardD4dsV5: VMSizeStandardD4dsV5Struct,
}

func GetCICandidateMasterVMSizes() []VMSize {
	var vmSizes = slices.Collect(maps.Keys(MinMasterVMSizes))
	return shuffler(vmSizes)
}

func GetCICandidateWorkerVMSizes() []VMSize {
	var vmSizes = slices.Collect(maps.Keys(MinWorkerVMSizes))
	return shuffler(vmSizes)
}
func shuffler(vmSizes []VMSize) []VMSize {
	rand.Shuffle(len(vmSizes), func(i, j int) {
		vmSizes[i], vmSizes[j] = vmSizes[j], vmSizes[i]
	})
	d2Count := 3
	if len(vmSizes) > d2Count {
		fallbacks := vmSizes[d2Count:]
		rand.Shuffle(len(fallbacks), func(i, j int) {
			fallbacks[i], fallbacks[j] = fallbacks[j], fallbacks[i]
		})
	}
	return vmSizes
}

// MinVMSizesForRole returns the minimum VM sizes for a role, sorted by core
// count (smallest first). This is used by test/CI/dev tooling.
func MinVMSizesForRole(vmRole VMRole) []VMSize {
	var m map[VMSize]VMSizeStruct
	switch vmRole {
	case VMRoleMaster:
		m = MinMasterVMSizes
	case VMRoleWorker:
		m = MinWorkerVMSizes
	default:
		return nil
	}

	type entry struct {
		size      VMSize
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

	result := make([]VMSize, len(entries))
	for i, e := range entries {
		result[i] = e.size
	}
	return result
}
