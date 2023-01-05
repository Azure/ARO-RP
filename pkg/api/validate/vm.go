package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

var supportedMasterVMSizes = map[api.VMSize]bool{
	// General purpose
	api.VMSizeStandardD8sV3:  true,
	api.VMSizeStandardD16sV3: true,
	api.VMSizeStandardD32sV3: true,
	// Memory optimized
	api.VMSizeStandardE64iV3:    true,
	api.VMSizeStandardE64isV3:   true,
	api.VMSizeStandardE80isV4:   true,
	api.VMSizeStandardE80idsV4:  true,
	api.VMSizeStandardE104iV5:   true,
	api.VMSizeStandardE104isV5:  true,
	api.VMSizeStandardE104idV5:  true,
	api.VMSizeStandardE104idsV5: true,
	// Compute optimized
	api.VMSizeStandardF72sV2: true,
	// Memory and storage optimized
	api.VMSizeStandardGS5: true,
	api.VMSizeStandardG5:  true,
	// Memory and compute optimized
	api.VMSizeStandardM128ms: true,
}

var supportedWorkerVMSizes = map[api.VMSize]bool{
	// General purpose
	api.VMSizeStandardD4asV4:  true,
	api.VMSizeStandardD8asV4:  true,
	api.VMSizeStandardD16asV4: true,
	api.VMSizeStandardD32asV4: true,
	api.VMSizeStandardD4sV3:   true,
	api.VMSizeStandardD8sV3:   true,
	api.VMSizeStandardD16sV3:  true,
	api.VMSizeStandardD32sV3:  true,
	// Memory optimized
	api.VMSizeStandardE4sV3:     true,
	api.VMSizeStandardE8sV3:     true,
	api.VMSizeStandardE16sV3:    true,
	api.VMSizeStandardE32sV3:    true,
	api.VMSizeStandardE64isV3:   true,
	api.VMSizeStandardE64iV3:    true,
	api.VMSizeStandardE80isV4:   true,
	api.VMSizeStandardE80idsV4:  true,
	api.VMSizeStandardE104iV5:   true,
	api.VMSizeStandardE104isV5:  true,
	api.VMSizeStandardE104idV5:  true,
	api.VMSizeStandardE104idsV5: true,
	// Compute optimized
	api.VMSizeStandardF4sV2:  true,
	api.VMSizeStandardF8sV2:  true,
	api.VMSizeStandardF16sV2: true,
	api.VMSizeStandardF32sV2: true,
	api.VMSizeStandardF72sV2: true,
	// Memory and storage optimized
	api.VMSizeStandardG5:  true,
	api.VMSizeStandardGS5: true,
	// Memory and compute optimized
	api.VMSizeStandardM128ms: true,
	// Storage optimized
	api.VMSizeStandardL4s:    true,
	api.VMSizeStandardL8s:    true,
	api.VMSizeStandardL16s:   true,
	api.VMSizeStandardL32s:   true,
	api.VMSizeStandardL8sV2:  true,
	api.VMSizeStandardL16sV2: true,
	api.VMSizeStandardL32sV2: true,
	api.VMSizeStandardL48sV2: true,
	api.VMSizeStandardL64sV2: true,
	// GPU
	api.VMSizeStandardNC4asT4V3:  true,
	api.VMSizeStandardNC8asT4V3:  true,
	api.VMSizeStandardNC16asT4V3: true,
	api.VMSizeStandardNC64asT4V3: true,
}

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, requiredD2sV3Workers, isMaster bool) bool {
	if isMaster {
		return supportedMasterVMSizes[vmSize]
	}

	if (supportedWorkerVMSizes[vmSize] && !requiredD2sV3Workers) ||
		(requiredD2sV3Workers && vmSize == api.VMSizeStandardD2sV3) {
		return true
	}

	return false
}
