package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// Public facing document which lists supported VM Sizes:
// https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4#supported-virtual-machine-sizes

// To add new instance types, needs Project Managment's involment and instructions are below.,
// https://github.com/Azure/ARO-RP/blob/master/docs/adding-new-instance-types.md

var SupportedMasterVmSizes = map[api.VMSize]api.VMSizeStruct{
	// General purpose
	api.VMSizeStandardD8sV3:  api.VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: api.VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: api.VMSizeStandardD32sV3Struct,

	api.VMSizeStandardD8sV4:  api.VMSizeStandardD8sV4Struct,
	api.VMSizeStandardD16sV4: api.VMSizeStandardD16sV4Struct,
	api.VMSizeStandardD32sV4: api.VMSizeStandardD32sV4Struct,

	api.VMSizeStandardD8sV5:  api.VMSizeStandardD8sV5Struct,
	api.VMSizeStandardD16sV5: api.VMSizeStandardD16sV5Struct,
	api.VMSizeStandardD32sV5: api.VMSizeStandardD32sV5Struct,

	api.VMSizeStandardD8asV4:  api.VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD16asV4: api.VMSizeStandardD16asV4Struct,
	api.VMSizeStandardD32asV4: api.VMSizeStandardD32asV4Struct,

	api.VMSizeStandardD8asV5:  api.VMSizeStandardD8asV5Struct,
	api.VMSizeStandardD16asV5: api.VMSizeStandardD16asV5Struct,
	api.VMSizeStandardD32asV5: api.VMSizeStandardD32asV5Struct,

	// Memory optimized
	api.VMSizeStandardE8sV3:  api.VMSizeStandardE8sV3Struct,
	api.VMSizeStandardE16sV3: api.VMSizeStandardE16sV3Struct,
	api.VMSizeStandardE32sV3: api.VMSizeStandardE32sV3Struct,

	api.VMSizeStandardE8sV4:  api.VMSizeStandardE8sV4Struct,
	api.VMSizeStandardE16sV4: api.VMSizeStandardE16sV4Struct,
	api.VMSizeStandardE20sV4: api.VMSizeStandardE20sV4Struct,
	api.VMSizeStandardE32sV4: api.VMSizeStandardE32sV4Struct,
	api.VMSizeStandardE48sV4: api.VMSizeStandardE48sV4Struct,
	api.VMSizeStandardE64sV4: api.VMSizeStandardE64sV4Struct,

	api.VMSizeStandardE8sV5:  api.VMSizeStandardE8sV5Struct,
	api.VMSizeStandardE16sV5: api.VMSizeStandardE16sV5Struct,
	api.VMSizeStandardE20sV5: api.VMSizeStandardE20sV5Struct,
	api.VMSizeStandardE32sV5: api.VMSizeStandardE32sV5Struct,
	api.VMSizeStandardE48sV5: api.VMSizeStandardE48sV5Struct,
	api.VMSizeStandardE64sV5: api.VMSizeStandardE64sV5Struct,
	api.VMSizeStandardE96sV5: api.VMSizeStandardE96sV5Struct,

	api.VMSizeStandardE4asV4:  api.VMSizeStandardE4asV4Struct,
	api.VMSizeStandardE8asV4:  api.VMSizeStandardE8asV4Struct,
	api.VMSizeStandardE16asV4: api.VMSizeStandardE16asV4Struct,
	api.VMSizeStandardE20asV4: api.VMSizeStandardE20asV4Struct,
	api.VMSizeStandardE32asV4: api.VMSizeStandardE32asV4Struct,
	api.VMSizeStandardE48asV4: api.VMSizeStandardE48asV4Struct,
	api.VMSizeStandardE64asV4: api.VMSizeStandardE64asV4Struct,
	api.VMSizeStandardE96asV4: api.VMSizeStandardE64asV4Struct,

	api.VMSizeStandardE8asV5:  api.VMSizeStandardE8asV5Struct,
	api.VMSizeStandardE16asV5: api.VMSizeStandardE16asV5Struct,
	api.VMSizeStandardE20asV5: api.VMSizeStandardE20asV5Struct,
	api.VMSizeStandardE32asV5: api.VMSizeStandardE32asV5Struct,
	api.VMSizeStandardE48asV5: api.VMSizeStandardE48asV5Struct,
	api.VMSizeStandardE64asV5: api.VMSizeStandardE64asV5Struct,
	api.VMSizeStandardE96asV5: api.VMSizeStandardE96asV5Struct,

	api.VMSizeStandardE64isV3:   api.VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE80isV4:   api.VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  api.VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104isV5:  api.VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idsV5: api.VMSizeStandardE104idsV5Struct,

	// Compute optimized
	api.VMSizeStandardF72sV2: api.VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: api.VMSizeStandardM128msStruct,
}

// Document support
var SupportedWorkerVmSizes = map[api.VMSize]api.VMSizeStruct{
	// General purpose
	api.VMSizeStandardD4sV3:  api.VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD8sV3:  api.VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: api.VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: api.VMSizeStandardD32sV3Struct,

	api.VMSizeStandardD4sV4:  api.VMSizeStandardD4sV4Struct,
	api.VMSizeStandardD8sV4:  api.VMSizeStandardD8sV4Struct,
	api.VMSizeStandardD16sV4: api.VMSizeStandardD16sV4Struct,
	api.VMSizeStandardD32sV4: api.VMSizeStandardD32sV4Struct,
	api.VMSizeStandardD64sV4: api.VMSizeStandardD64sV4Struct,
	api.VMSizeStandardD96sV4: api.VMSizeStandardD96sV4Struct,

	api.VMSizeStandardD4sV5:  api.VMSizeStandardD4sV5Struct,
	api.VMSizeStandardD8sV5:  api.VMSizeStandardD8sV5Struct,
	api.VMSizeStandardD16sV5: api.VMSizeStandardD16sV5Struct,
	api.VMSizeStandardD32sV5: api.VMSizeStandardD32sV5Struct,
	api.VMSizeStandardD64sV5: api.VMSizeStandardD64sV5Struct,
	api.VMSizeStandardD96sV5: api.VMSizeStandardD96sV5Struct,

	api.VMSizeStandardD4asV4:  api.VMSizeStandardD4asV4Struct,
	api.VMSizeStandardD8asV4:  api.VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD16asV4: api.VMSizeStandardD16asV4Struct,
	api.VMSizeStandardD32asV4: api.VMSizeStandardD32asV4Struct,
	api.VMSizeStandardD64asV4: api.VMSizeStandardD64asV4Struct,
	api.VMSizeStandardD96asV4: api.VMSizeStandardD96asV4Struct,

	api.VMSizeStandardD4asV5:  api.VMSizeStandardD4asV5Struct,
	api.VMSizeStandardD8asV5:  api.VMSizeStandardD8asV5Struct,
	api.VMSizeStandardD16asV5: api.VMSizeStandardD16asV5Struct,
	api.VMSizeStandardD32asV5: api.VMSizeStandardD32asV5Struct,
	api.VMSizeStandardD64asV5: api.VMSizeStandardD64asV5Struct,
	api.VMSizeStandardD96asV5: api.VMSizeStandardD96asV5Struct,

	// Memory optimized
	api.VMSizeStandardE4sV3:  api.VMSizeStandardE4sV3Struct,
	api.VMSizeStandardE8sV3:  api.VMSizeStandardE8sV3Struct,
	api.VMSizeStandardE16sV3: api.VMSizeStandardE16sV3Struct,
	api.VMSizeStandardE32sV3: api.VMSizeStandardE32sV3Struct,

	api.VMSizeStandardE2sV4:  api.VMSizeStandardE2sV4Struct,
	api.VMSizeStandardE4sV4:  api.VMSizeStandardE4sV4Struct,
	api.VMSizeStandardE8sV4:  api.VMSizeStandardE8sV4Struct,
	api.VMSizeStandardE16sV4: api.VMSizeStandardE16sV4Struct,
	api.VMSizeStandardE20sV4: api.VMSizeStandardE20sV4Struct,
	api.VMSizeStandardE32sV4: api.VMSizeStandardE32sV4Struct,
	api.VMSizeStandardE48sV4: api.VMSizeStandardE48sV4Struct,
	api.VMSizeStandardE64sV4: api.VMSizeStandardE64sV4Struct,
	api.VMSizeStandardE96sV4: api.VMSizeStandardE96sV4Struct,

	api.VMSizeStandardE2sV5:  api.VMSizeStandardE2sV5Struct,
	api.VMSizeStandardE4sV5:  api.VMSizeStandardE4sV5Struct,
	api.VMSizeStandardE8sV5:  api.VMSizeStandardE8sV5Struct,
	api.VMSizeStandardE16sV5: api.VMSizeStandardE16sV5Struct,
	api.VMSizeStandardE20sV5: api.VMSizeStandardE20sV5Struct,
	api.VMSizeStandardE32sV5: api.VMSizeStandardE32sV5Struct,
	api.VMSizeStandardE48sV5: api.VMSizeStandardE48sV5Struct,
	api.VMSizeStandardE64sV5: api.VMSizeStandardE64sV5Struct,
	api.VMSizeStandardE96sV5: api.VMSizeStandardE96sV5Struct,

	api.VMSizeStandardE4asV4:  api.VMSizeStandardE4asV4Struct,
	api.VMSizeStandardE8asV4:  api.VMSizeStandardE8asV4Struct,
	api.VMSizeStandardE16asV4: api.VMSizeStandardE16asV4Struct,
	api.VMSizeStandardE20asV4: api.VMSizeStandardE20asV4Struct,
	api.VMSizeStandardE32asV4: api.VMSizeStandardE32asV4Struct,
	api.VMSizeStandardE48asV4: api.VMSizeStandardE48asV4Struct,
	api.VMSizeStandardE64asV4: api.VMSizeStandardE64asV4Struct,
	api.VMSizeStandardE96asV4: api.VMSizeStandardE64asV4Struct,

	api.VMSizeStandardE8asV5:  api.VMSizeStandardE8asV5Struct,
	api.VMSizeStandardE16asV5: api.VMSizeStandardE16asV5Struct,
	api.VMSizeStandardE20asV5: api.VMSizeStandardE20asV5Struct,
	api.VMSizeStandardE32asV5: api.VMSizeStandardE32asV5Struct,
	api.VMSizeStandardE48asV5: api.VMSizeStandardE48asV5Struct,
	api.VMSizeStandardE64asV5: api.VMSizeStandardE64asV5Struct,
	api.VMSizeStandardE96asV5: api.VMSizeStandardE96asV5Struct,

	api.VMSizeStandardE64isV3:   api.VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE80isV4:   api.VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  api.VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104isV5:  api.VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idsV5: api.VMSizeStandardE104idsV5Struct,

	// Compute optimized
	api.VMSizeStandardF4sV2:  api.VMSizeStandardF4sV2Struct,
	api.VMSizeStandardF8sV2:  api.VMSizeStandardF8sV2Struct,
	api.VMSizeStandardF16sV2: api.VMSizeStandardF16sV2Struct,
	api.VMSizeStandardF32sV2: api.VMSizeStandardF32sV2Struct,
	api.VMSizeStandardF72sV2: api.VMSizeStandardF72sV2Struct,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: api.VMSizeStandardM128msStruct,

	// Storage optimized
	api.VMSizeStandardL4s:  api.VMSizeStandardL4sStruct,
	api.VMSizeStandardL8s:  api.VMSizeStandardL8sStruct,
	api.VMSizeStandardL16s: api.VMSizeStandardL16sStruct,
	api.VMSizeStandardL32s: api.VMSizeStandardL32sStruct,

	api.VMSizeStandardL8sV2:  api.VMSizeStandardL8sV2Struct,
	api.VMSizeStandardL16sV2: api.VMSizeStandardL16sV2Struct,
	api.VMSizeStandardL32sV2: api.VMSizeStandardL32sV2Struct,
	api.VMSizeStandardL48sV2: api.VMSizeStandardL48sV2Struct,
	api.VMSizeStandardL64sV2: api.VMSizeStandardL64sV2Struct,

	api.VMSizeStandardL8sV3:  api.VMSizeStandardL8sV3Struct,
	api.VMSizeStandardL16sV3: api.VMSizeStandardL16sV3Struct,
	api.VMSizeStandardL32sV3: api.VMSizeStandardL32sV3Struct,
	api.VMSizeStandardL48sV3: api.VMSizeStandardL48sV3Struct,
	api.VMSizeStandardL64sV3: api.VMSizeStandardL64sV3Struct,

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	api.VMSizeStandardNC4asT4V3:  api.VMSizeStandardNC4asT4V3Struct,
	api.VMSizeStandardNC8asT4V3:  api.VMSizeStandardNC8asT4V3Struct,
	api.VMSizeStandardNC16asT4V3: api.VMSizeStandardNC16asT4V3Struct,
	api.VMSizeStandardNC64asT4V3: api.VMSizeStandardNC64asT4V3Struct,

	api.VMSizeStandardNC6sV3:   api.VMSizeStandardNC6sV3Struct,
	api.VMSizeStandardNC12sV3:  api.VMSizeStandardNC12sV3Struct,
	api.VMSizeStandardNC24sV3:  api.VMSizeStandardNC24sV3Struct,
	api.VMSizeStandardNC24rsV3: api.VMSizeStandardNC24rsV3Struct,
}

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, requiredD2sV3Workers, isMaster bool) bool {
	if isMaster {
		_, supportedAsMaster := SupportedMasterVmSizes[vmSize]
		return supportedAsMaster
	}

	if requiredD2sV3Workers && vmSize != api.VMSizeStandardD2sV3 {
		return false
	}

	_, supportedAsWorker := SupportedWorkerVmSizes[vmSize]
	if supportedAsWorker || (requiredD2sV3Workers && vmSize == api.VMSizeStandardD2sV3) {
		return true
	}

	return false
}

func VMSizeFromName(vmSize api.VMSize) (api.VMSizeStruct, bool) {
	//this is for development purposes only
	if vmSize == api.VMSizeStandardD2sV3 {
		return api.VMSizeStandardD2sV3Struct, true
	}

	if size, ok := SupportedWorkerVmSizes[vmSize]; ok {
		return size, true
	}

	if size, ok := SupportedMasterVmSizes[vmSize]; ok {
		return size, true
	}
	return api.VMSizeStruct{}, false
}
