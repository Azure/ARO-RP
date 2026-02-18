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

var SupportedMasterVMSizes = map[VMSize]VMSizeStruct{
	// General purpose
	VMSizeStandardD8sV3:  vmSizeStandardD8sV3Struct,
	VMSizeStandardD16sV3: vmSizeStandardD16sV3Struct,
	VMSizeStandardD32sV3: vmSizeStandardD32sV3Struct,

	VMSizeStandardD8sV4:  vmSizeStandardD8sV4Struct,
	VMSizeStandardD16sV4: vmSizeStandardD16sV4Struct,
	VMSizeStandardD32sV4: vmSizeStandardD32sV4Struct,

	VMSizeStandardD8sV5:  vmSizeStandardD8sV5Struct,
	VMSizeStandardD16sV5: vmSizeStandardD16sV5Struct,
	VMSizeStandardD32sV5: vmSizeStandardD32sV5Struct,

	VMSizeStandardD8asV4:  vmSizeStandardD8asV4Struct,
	VMSizeStandardD16asV4: vmSizeStandardD16asV4Struct,
	VMSizeStandardD32asV4: vmSizeStandardD32asV4Struct,

	VMSizeStandardD8asV5:  vmSizeStandardD8asV5Struct,
	VMSizeStandardD16asV5: vmSizeStandardD16asV5Struct,
	VMSizeStandardD32asV5: vmSizeStandardD32asV5Struct,

	VMSizeStandardD8dsV5:  vmSizeStandardD8dsV5Struct,
	VMSizeStandardD16dsV5: vmSizeStandardD16dsV5Struct,
	VMSizeStandardD32dsV5: vmSizeStandardD32dsV5Struct,

	// Memory optimized
	VMSizeStandardE8sV3:  vmSizeStandardE8sV3Struct,
	VMSizeStandardE16sV3: vmSizeStandardE16sV3Struct,
	VMSizeStandardE32sV3: vmSizeStandardE32sV3Struct,

	VMSizeStandardE8sV4:  vmSizeStandardE8sV4Struct,
	VMSizeStandardE16sV4: vmSizeStandardE16sV4Struct,
	VMSizeStandardE20sV4: vmSizeStandardE20sV4Struct,
	VMSizeStandardE32sV4: vmSizeStandardE32sV4Struct,
	VMSizeStandardE48sV4: vmSizeStandardE48sV4Struct,
	VMSizeStandardE64sV4: vmSizeStandardE64sV4Struct,

	VMSizeStandardE8sV5:  vmSizeStandardE8sV5Struct,
	VMSizeStandardE16sV5: vmSizeStandardE16sV5Struct,
	VMSizeStandardE20sV5: vmSizeStandardE20sV5Struct,
	VMSizeStandardE32sV5: vmSizeStandardE32sV5Struct,
	VMSizeStandardE48sV5: vmSizeStandardE48sV5Struct,
	VMSizeStandardE64sV5: vmSizeStandardE64sV5Struct,
	VMSizeStandardE96sV5: vmSizeStandardE96sV5Struct,

	VMSizeStandardE4asV4:  vmSizeStandardE4asV4Struct,
	VMSizeStandardE8asV4:  vmSizeStandardE8asV4Struct,
	VMSizeStandardE16asV4: vmSizeStandardE16asV4Struct,
	VMSizeStandardE20asV4: vmSizeStandardE20asV4Struct,
	VMSizeStandardE32asV4: vmSizeStandardE32asV4Struct,
	VMSizeStandardE48asV4: vmSizeStandardE48asV4Struct,
	VMSizeStandardE64asV4: vmSizeStandardE64asV4Struct,
	VMSizeStandardE96asV4: vmSizeStandardE96asV4Struct,

	VMSizeStandardE8asV5:  vmSizeStandardE8asV5Struct,
	VMSizeStandardE16asV5: vmSizeStandardE16asV5Struct,
	VMSizeStandardE20asV5: vmSizeStandardE20asV5Struct,
	VMSizeStandardE32asV5: vmSizeStandardE32asV5Struct,
	VMSizeStandardE48asV5: vmSizeStandardE48asV5Struct,
	VMSizeStandardE64asV5: vmSizeStandardE64asV5Struct,
	VMSizeStandardE96asV5: vmSizeStandardE96asV5Struct,

	VMSizeStandardE64isV3:   vmSizeStandardE64isV3Struct,
	VMSizeStandardE80isV4:   vmSizeStandardE80isV4Struct,
	VMSizeStandardE80idsV4:  vmSizeStandardE80idsV4Struct,
	VMSizeStandardE104isV5:  vmSizeStandardE104isV5Struct,
	VMSizeStandardE104idsV5: vmSizeStandardE104idsV5Struct,

	// Compute optimized
	VMSizeStandardF72sV2: vmSizeStandardF72sV2Struct,

	// Memory and compute optimized
	VMSizeStandardM128ms: vmSizeStandardM128msStruct,

	VMSizeStandardD4sV6:  vmSizeStandardD4sV6Struct,
	VMSizeStandardD8sV6:  vmSizeStandardD8sV6Struct,
	VMSizeStandardD16sV6: vmSizeStandardD16sV6Struct,
	VMSizeStandardD32sV6: vmSizeStandardD32sV6Struct,
	VMSizeStandardD48sV6: vmSizeStandardD48sV6Struct,
	VMSizeStandardD64sV6: vmSizeStandardD64sV6Struct,
	VMSizeStandardD96sV6: vmSizeStandardD96sV6Struct,

	VMSizeStandardD4dsV6:  vmSizeStandardD4dsV6Struct,
	VMSizeStandardD8dsV6:  vmSizeStandardD8dsV6Struct,
	VMSizeStandardD16dsV6: vmSizeStandardD16dsV6Struct,
	VMSizeStandardD32dsV6: vmSizeStandardD32dsV6Struct,
	VMSizeStandardD48dsV6: vmSizeStandardD48dsV6Struct,
	VMSizeStandardD64dsV6: vmSizeStandardD64dsV6Struct,
	VMSizeStandardD96dsV6: vmSizeStandardD96dsV6Struct,
}

var SupportedWorkerVMSizes = map[VMSize]VMSizeStruct{
	// General purpose
	VMSizeStandardD4sV3:  vmSizeStandardD4sV3Struct,
	VMSizeStandardD8sV3:  vmSizeStandardD8sV3Struct,
	VMSizeStandardD16sV3: vmSizeStandardD16sV3Struct,
	VMSizeStandardD32sV3: vmSizeStandardD32sV3Struct,

	VMSizeStandardD4sV4:  vmSizeStandardD4sV4Struct,
	VMSizeStandardD8sV4:  vmSizeStandardD8sV4Struct,
	VMSizeStandardD16sV4: vmSizeStandardD16sV4Struct,
	VMSizeStandardD32sV4: vmSizeStandardD32sV4Struct,
	VMSizeStandardD64sV4: vmSizeStandardD64sV4Struct,

	VMSizeStandardD4sV5:  vmSizeStandardD4sV5Struct,
	VMSizeStandardD8sV5:  vmSizeStandardD8sV5Struct,
	VMSizeStandardD16sV5: vmSizeStandardD16sV5Struct,
	VMSizeStandardD32sV5: vmSizeStandardD32sV5Struct,
	VMSizeStandardD64sV5: vmSizeStandardD64sV5Struct,
	VMSizeStandardD96sV5: vmSizeStandardD96sV5Struct,

	VMSizeStandardD4asV4:  vmSizeStandardD4asV4Struct,
	VMSizeStandardD8asV4:  vmSizeStandardD8asV4Struct,
	VMSizeStandardD16asV4: vmSizeStandardD16asV4Struct,
	VMSizeStandardD32asV4: vmSizeStandardD32asV4Struct,
	VMSizeStandardD64asV4: vmSizeStandardD64asV4Struct,
	VMSizeStandardD96asV4: vmSizeStandardD96asV4Struct,

	VMSizeStandardD4asV5:  vmSizeStandardD4asV5Struct,
	VMSizeStandardD8asV5:  vmSizeStandardD8asV5Struct,
	VMSizeStandardD16asV5: vmSizeStandardD16asV5Struct,
	VMSizeStandardD32asV5: vmSizeStandardD32asV5Struct,
	VMSizeStandardD64asV5: vmSizeStandardD64asV5Struct,
	VMSizeStandardD96asV5: vmSizeStandardD96asV5Struct,

	VMSizeStandardD4dsV5:  vmSizeStandardD4dsV5Struct,
	VMSizeStandardD8dsV5:  vmSizeStandardD8dsV5Struct,
	VMSizeStandardD16dsV5: vmSizeStandardD16dsV5Struct,
	VMSizeStandardD32dsV5: vmSizeStandardD32dsV5Struct,
	VMSizeStandardD64dsV5: vmSizeStandardD64dsV5Struct,
	VMSizeStandardD96dsV5: vmSizeStandardD96dsV5Struct,

	// Memory optimized
	VMSizeStandardE4sV3:  vmSizeStandardE4sV3Struct,
	VMSizeStandardE8sV3:  vmSizeStandardE8sV3Struct,
	VMSizeStandardE16sV3: vmSizeStandardE16sV3Struct,
	VMSizeStandardE32sV3: vmSizeStandardE32sV3Struct,

	VMSizeStandardE2sV4:  vmSizeStandardE2sV4Struct,
	VMSizeStandardE4sV4:  vmSizeStandardE4sV4Struct,
	VMSizeStandardE8sV4:  vmSizeStandardE8sV4Struct,
	VMSizeStandardE16sV4: vmSizeStandardE16sV4Struct,
	VMSizeStandardE20sV4: vmSizeStandardE20sV4Struct,
	VMSizeStandardE32sV4: vmSizeStandardE32sV4Struct,
	VMSizeStandardE48sV4: vmSizeStandardE48sV4Struct,
	VMSizeStandardE64sV4: vmSizeStandardE64sV4Struct,

	VMSizeStandardE2sV5:  vmSizeStandardE2sV5Struct,
	VMSizeStandardE4sV5:  vmSizeStandardE4sV5Struct,
	VMSizeStandardE8sV5:  vmSizeStandardE8sV5Struct,
	VMSizeStandardE16sV5: vmSizeStandardE16sV5Struct,
	VMSizeStandardE20sV5: vmSizeStandardE20sV5Struct,
	VMSizeStandardE32sV5: vmSizeStandardE32sV5Struct,
	VMSizeStandardE48sV5: vmSizeStandardE48sV5Struct,
	VMSizeStandardE64sV5: vmSizeStandardE64sV5Struct,
	VMSizeStandardE96sV5: vmSizeStandardE96sV5Struct,

	VMSizeStandardE4asV4:  vmSizeStandardE4asV4Struct,
	VMSizeStandardE8asV4:  vmSizeStandardE8asV4Struct,
	VMSizeStandardE16asV4: vmSizeStandardE16asV4Struct,
	VMSizeStandardE20asV4: vmSizeStandardE20asV4Struct,
	VMSizeStandardE32asV4: vmSizeStandardE32asV4Struct,
	VMSizeStandardE48asV4: vmSizeStandardE48asV4Struct,
	VMSizeStandardE64asV4: vmSizeStandardE64asV4Struct,
	VMSizeStandardE96asV4: vmSizeStandardE96asV4Struct,

	VMSizeStandardE8asV5:  vmSizeStandardE8asV5Struct,
	VMSizeStandardE16asV5: vmSizeStandardE16asV5Struct,
	VMSizeStandardE20asV5: vmSizeStandardE20asV5Struct,
	VMSizeStandardE32asV5: vmSizeStandardE32asV5Struct,
	VMSizeStandardE48asV5: vmSizeStandardE48asV5Struct,
	VMSizeStandardE64asV5: vmSizeStandardE64asV5Struct,
	VMSizeStandardE96asV5: vmSizeStandardE96asV5Struct,

	VMSizeStandardE64isV3:   vmSizeStandardE64isV3Struct,
	VMSizeStandardE80isV4:   vmSizeStandardE80isV4Struct,
	VMSizeStandardE80idsV4:  vmSizeStandardE80idsV4Struct,
	VMSizeStandardE104isV5:  vmSizeStandardE104isV5Struct,
	VMSizeStandardE104idsV5: vmSizeStandardE104idsV5Struct,

	// Compute optimized
	VMSizeStandardF4sV2:  vmSizeStandardF4sV2Struct,
	VMSizeStandardF8sV2:  vmSizeStandardF8sV2Struct,
	VMSizeStandardF16sV2: vmSizeStandardF16sV2Struct,
	VMSizeStandardF32sV2: vmSizeStandardF32sV2Struct,
	VMSizeStandardF72sV2: vmSizeStandardF72sV2Struct,

	// Memory and compute optimized
	VMSizeStandardM128ms: vmSizeStandardM128msStruct,

	// Storage optimized
	VMSizeStandardL4s:  vmSizeStandardL4sStruct,
	VMSizeStandardL8s:  vmSizeStandardL8sStruct,
	VMSizeStandardL16s: vmSizeStandardL16sStruct,
	VMSizeStandardL32s: vmSizeStandardL32sStruct,

	VMSizeStandardL8sV2:  vmSizeStandardL8sV2Struct,
	VMSizeStandardL16sV2: vmSizeStandardL16sV2Struct,
	VMSizeStandardL32sV2: vmSizeStandardL32sV2Struct,
	VMSizeStandardL48sV2: vmSizeStandardL48sV2Struct,
	VMSizeStandardL64sV2: vmSizeStandardL64sV2Struct,

	VMSizeStandardL8sV3:  vmSizeStandardL8sV3Struct,
	VMSizeStandardL16sV3: vmSizeStandardL16sV3Struct,
	VMSizeStandardL32sV3: vmSizeStandardL32sV3Struct,
	VMSizeStandardL48sV3: vmSizeStandardL48sV3Struct,
	VMSizeStandardL64sV3: vmSizeStandardL64sV3Struct,

	VMSizeStandardL4sV4:  vmSizeStandardL4sV4Struct,
	VMSizeStandardL8sV4:  vmSizeStandardL8sV4Struct,
	VMSizeStandardL16sV4: vmSizeStandardL16sV4Struct,
	VMSizeStandardL32sV4: vmSizeStandardL32sV4Struct,
	VMSizeStandardL48sV4: vmSizeStandardL48sV4Struct,
	VMSizeStandardL64sV4: vmSizeStandardL64sV4Struct,
	VMSizeStandardL80sV4: vmSizeStandardL80sV4Struct,

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	VMSizeStandardNC4asT4V3:  vmSizeStandardNC4asT4V3Struct,
	VMSizeStandardNC8asT4V3:  vmSizeStandardNC8asT4V3Struct,
	VMSizeStandardNC16asT4V3: vmSizeStandardNC16asT4V3Struct,
	VMSizeStandardNC64asT4V3: vmSizeStandardNC64asT4V3Struct,

	VMSizeStandardNC6sV3:   vmSizeStandardNC6sV3Struct,
	VMSizeStandardNC12sV3:  vmSizeStandardNC12sV3Struct,
	VMSizeStandardNC24sV3:  vmSizeStandardNC24sV3Struct,
	VMSizeStandardNC24rsV3: vmSizeStandardNC24rsV3Struct,

	VMSizeStandardD4sV6:  vmSizeStandardD4sV6Struct,
	VMSizeStandardD8sV6:  vmSizeStandardD8sV6Struct,
	VMSizeStandardD16sV6: vmSizeStandardD16sV6Struct,
	VMSizeStandardD32sV6: vmSizeStandardD32sV6Struct,
	VMSizeStandardD48sV6: vmSizeStandardD48sV6Struct,
	VMSizeStandardD64sV6: vmSizeStandardD64sV6Struct,
	VMSizeStandardD96sV6: vmSizeStandardD96sV6Struct,

	VMSizeStandardD4dsV6:  vmSizeStandardD4dsV6Struct,
	VMSizeStandardD8dsV6:  vmSizeStandardD8dsV6Struct,
	VMSizeStandardD16dsV6: vmSizeStandardD16dsV6Struct,
	VMSizeStandardD32dsV6: vmSizeStandardD32dsV6Struct,
	VMSizeStandardD48dsV6: vmSizeStandardD48dsV6Struct,
	VMSizeStandardD64dsV6: vmSizeStandardD64dsV6Struct,
	VMSizeStandardD96dsV6: vmSizeStandardD96dsV6Struct,

	VMSizeStandardD4lsV6:  vmSizeStandardD4lsV6Struct,
	VMSizeStandardD8lsV6:  vmSizeStandardD8lsV6Struct,
	VMSizeStandardD16lsV6: vmSizeStandardD16lsV6Struct,
	VMSizeStandardD32lsV6: vmSizeStandardD32lsV6Struct,
	VMSizeStandardD48lsV6: vmSizeStandardD48lsV6Struct,
	VMSizeStandardD64lsV6: vmSizeStandardD64lsV6Struct,
	VMSizeStandardD96lsV6: vmSizeStandardD96lsV6Struct,

	VMSizeStandardD4ldsV6:  vmSizeStandardD4ldsV6Struct,
	VMSizeStandardD8ldsV6:  vmSizeStandardD8ldsV6Struct,
	VMSizeStandardD16ldsV6: vmSizeStandardD16ldsV6Struct,
	VMSizeStandardD32ldsV6: vmSizeStandardD32ldsV6Struct,
	VMSizeStandardD48ldsV6: vmSizeStandardD48ldsV6Struct,
	VMSizeStandardD64ldsV6: vmSizeStandardD64ldsV6Struct,
	VMSizeStandardD96ldsV6: vmSizeStandardD96ldsV6Struct,
}

var supportedMasterVMSizesForInternalUser = map[VMSize]VMSizeStruct{
	VMSizeStandardD4sV3: vmSizeStandardD4sV3Struct,
	VMSizeStandardD4sV4: vmSizeStandardD4sV4Struct,
	VMSizeStandardD4sV5: vmSizeStandardD4sV5Struct,
	VMSizeStandardD4sV6: vmSizeStandardD4sV6Struct,
}

var supportedWorkerVMSizesForInternalUser = map[VMSize]VMSizeStruct{
	VMSizeStandardD2sV3: vmSizeStandardD2sV3Struct,
	VMSizeStandardD2sV4: vmSizeStandardD2sV4Struct,
	VMSizeStandardD2sV5: vmSizeStandardD2sV5Struct,
	VMSizeStandardD2sV6: vmSizeStandardD2sV6Struct,
}

func init() {
	maps.Copy(supportedMasterVMSizesForInternalUser, SupportedMasterVMSizes)
	maps.Copy(supportedWorkerVMSizesForInternalUser, SupportedWorkerVMSizes)
}

// TODO: MAITIU - Choose correct sizes

// minMasterVMSizes contains the smallest supported master VM size for each
// general-purpose D-series family. Used by test/CI/dev tooling to select
// cost-effective sizes while spreading quota across families.
var minMasterVMSizes = map[VMSize]VMSizeStruct{
	VMSizeStandardD8sV3:  vmSizeStandardD8sV3Struct,
	VMSizeStandardD8sV4:  vmSizeStandardD8sV4Struct,
	VMSizeStandardD8sV5:  vmSizeStandardD8sV5Struct,
	VMSizeStandardD8asV4: vmSizeStandardD8asV4Struct,
	VMSizeStandardD8asV5: vmSizeStandardD8asV5Struct,
	VMSizeStandardD8dsV5: vmSizeStandardD8dsV5Struct,
}

// minWorkerVMSizes contains the smallest supported worker VM size for each
// general-purpose D-series family.
var minWorkerVMSizes = map[VMSize]VMSizeStruct{
	VMSizeStandardD2sV5:  vmSizeStandardD2sV5Struct,
	VMSizeStandardD4sV3:  vmSizeStandardD4sV3Struct,
	VMSizeStandardD4sV4:  vmSizeStandardD4sV4Struct,
	VMSizeStandardD4sV5:  vmSizeStandardD4sV5Struct,
	VMSizeStandardD4asV4: vmSizeStandardD4asV4Struct,
	VMSizeStandardD4asV5: vmSizeStandardD4asV5Struct,
	VMSizeStandardD4dsV5: vmSizeStandardD4dsV5Struct,
}

// LookupVMSize returns the VMSizeStruct for a given VMSize by searching
// all supported size maps (including internal-user sizes).
func LookupVMSize(vmSize VMSize) (VMSizeStruct, bool) {
	if s, ok := SupportedWorkerVMSizes[vmSize]; ok {
		return s, true
	}
	if s, ok := SupportedMasterVMSizes[vmSize]; ok {
		return s, true
	}
	if s, ok := supportedWorkerVMSizesForInternalUser[vmSize]; ok {
		return s, true
	}
	if s, ok := supportedMasterVMSizesForInternalUser[vmSize]; ok {
		return s, true
	}
	return VMSizeStruct{}, false
}

func GetCICandidateMasterVMSizes() []VMSize {
	vmSizes := slices.Collect(maps.Keys(minMasterVMSizes))
	return shuffler(vmSizes)
}

func GetCICandidateWorkerVMSizes() []VMSize {
	vmSizes := slices.Collect(maps.Keys(minWorkerVMSizes))
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

// minVMSizesForRole returns the minimum VM sizes for a role, sorted by core
// count (smallest first). This is used by test/CI/dev tooling.
func minVMSizesForRole(vmRole VMRole) []VMSize {
	var m map[VMSize]VMSizeStruct
	switch vmRole {
	case VMRoleMaster:
		m = minMasterVMSizes
	case VMRoleWorker:
		m = minWorkerVMSizes
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
