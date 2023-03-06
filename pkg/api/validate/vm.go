package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

var SupportedMasterVmSizes = map[api.VMSize]api.VMSizeStruct{
	// General purpose
	api.VMSizeStandardD8sV3:  api.VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: api.VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: api.VMSizeStandardD32sV3Struct,

	// Memory optimized
	api.VMSizeStandardE64isV3:   api.VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE64iV3:    api.VMSizeStandardE64iV3Struct,
	api.VMSizeStandardE80isV4:   api.VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  api.VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104iV5:   api.VMSizeStandardE104iV5Struct,
	api.VMSizeStandardE104isV5:  api.VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idV5:  api.VMSizeStandardE104idV5Struct,
	api.VMSizeStandardE104idsV5: api.VMSizeStandardE104idsV5Struct,

	// Compute optimized
	api.VMSizeStandardF72sV2: api.VMSizeStandardF72sV2Struct,

	// Memory and storage optimized
	api.VMSizeStandardG5:  api.VMSizeStandardG5Struct,
	api.VMSizeStandardGS5: api.VMSizeStandardGS5Struct,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: api.VMSizeStandardM128msStruct,
}

var supportedWorkerVmSizes = map[api.VMSize]api.VMSizeStruct{
	api.VMSizeStandardD4asV4:  api.VMSizeStandardD4asV4Struct,
	api.VMSizeStandardD8asV4:  api.VMSizeStandardD8asV4Struct,
	api.VMSizeStandardD16asV4: api.VMSizeStandardD16asV4Struct,
	api.VMSizeStandardD32asV4: api.VMSizeStandardD32asV4Struct,

	api.VMSizeStandardD4sV3:  api.VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD8sV3:  api.VMSizeStandardD8sV3Struct,
	api.VMSizeStandardD16sV3: api.VMSizeStandardD16sV3Struct,
	api.VMSizeStandardD32sV3: api.VMSizeStandardD32sV3Struct,

	api.VMSizeStandardE4sV3:     api.VMSizeStandardE4sV3Struct,
	api.VMSizeStandardE8sV3:     api.VMSizeStandardE8sV3Struct,
	api.VMSizeStandardE16sV3:    api.VMSizeStandardE16sV3Struct,
	api.VMSizeStandardE32sV3:    api.VMSizeStandardE32sV3Struct,
	api.VMSizeStandardE64isV3:   api.VMSizeStandardE64isV3Struct,
	api.VMSizeStandardE64iV3:    api.VMSizeStandardE64iV3Struct,
	api.VMSizeStandardE80isV4:   api.VMSizeStandardE80isV4Struct,
	api.VMSizeStandardE80idsV4:  api.VMSizeStandardE80idsV4Struct,
	api.VMSizeStandardE104iV5:   api.VMSizeStandardE104iV5Struct,
	api.VMSizeStandardE104isV5:  api.VMSizeStandardE104isV5Struct,
	api.VMSizeStandardE104idV5:  api.VMSizeStandardE104idV5Struct,
	api.VMSizeStandardE104idsV5: api.VMSizeStandardE104idsV5Struct,

	api.VMSizeStandardF4sV2:  api.VMSizeStandardF4sV2Struct,
	api.VMSizeStandardF8sV2:  api.VMSizeStandardF8sV2Struct,
	api.VMSizeStandardF16sV2: api.VMSizeStandardF16sV2Struct,
	api.VMSizeStandardF32sV2: api.VMSizeStandardF32sV2Struct,
	api.VMSizeStandardF72sV2: api.VMSizeStandardF72sV2Struct,

	api.VMSizeStandardM128ms: api.VMSizeStandardM128msStruct,
	api.VMSizeStandardG5:     api.VMSizeStandardG5Struct,
	api.VMSizeStandardGS5:    api.VMSizeStandardGS5Struct,

	api.VMSizeStandardL4s:    api.VMSizeStandardL4sStruct,
	api.VMSizeStandardL8s:    api.VMSizeStandardL8sStruct,
	api.VMSizeStandardL16s:   api.VMSizeStandardL16sStruct,
	api.VMSizeStandardL32s:   api.VMSizeStandardL32sStruct,
	api.VMSizeStandardL8sV2:  api.VMSizeStandardL8sV2Struct,
	api.VMSizeStandardL16sV2: api.VMSizeStandardL16sV2Struct,
	api.VMSizeStandardL32sV2: api.VMSizeStandardL32sV2Struct,
	api.VMSizeStandardL48sV2: api.VMSizeStandardL48sV2Struct,
	api.VMSizeStandardL64sV2: api.VMSizeStandardL64sV2Struct,

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

	_, supportedAsWorker := supportedWorkerVmSizes[vmSize]
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

	if size, ok := supportedWorkerVmSizes[vmSize]; ok {
		return size, true
	}

	if size, ok := SupportedMasterVmSizes[vmSize]; ok {
		return size, true
	}
	return api.VMSizeStruct{}, false
}
