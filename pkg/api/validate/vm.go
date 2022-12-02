package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type VMSize struct {
	CoreCount int
	Family    string
}

const (
	standardDSv3    = "standardDSv3Family"
	standardDASv4   = "standardDASv4Family"
	standardESv3    = "standardESv3Family"
	standardEISv4   = "standardEISv4Family"
	standardEIDSv4  = "standardEIDSv4Family"
	standardEIv5    = "standardEIv5Family"
	standardEISv5   = "standardEISv5Family"
	standardEIDSv5  = "standardEIDSv5Family"
	standardEIDv5   = "standardEIDv5Family"
	standardFSv2    = "standardFSv2Family"
	standardMS      = "standardMSFamily"
	standardGFamily = "standardGFamily"
	standardLSv2    = "standardLsv2Family"
	standardNCAS    = "Standard NCASv3_T4 Family"
)

var (
	VMSizeStandardD2sV3 = VMSize{CoreCount: 2, Family: standardDSv3}

	VMSizeStandardD4asV4  = VMSize{CoreCount: 4, Family: standardDASv4}
	VMSizeStandardD8asV4  = VMSize{CoreCount: 8, Family: standardDASv4}
	VMSizeStandardD16asV4 = VMSize{CoreCount: 16, Family: standardDASv4}
	VMSizeStandardD32asV4 = VMSize{CoreCount: 32, Family: standardDASv4}

	VMSizeStandardD4sV3  = VMSize{CoreCount: 4, Family: standardDSv3}
	VMSizeStandardD8sV3  = VMSize{CoreCount: 8, Family: standardDSv3}
	VMSizeStandardD16sV3 = VMSize{CoreCount: 16, Family: standardDSv3}
	VMSizeStandardD32sV3 = VMSize{CoreCount: 32, Family: standardDSv3}

	VMSizeStandardE4sV3     = VMSize{CoreCount: 4, Family: standardESv3}
	VMSizeStandardE8sV3     = VMSize{CoreCount: 8, Family: standardESv3}
	VMSizeStandardE16sV3    = VMSize{CoreCount: 16, Family: standardESv3}
	VMSizeStandardE32sV3    = VMSize{CoreCount: 32, Family: standardESv3}
	VMSizeStandardE64isV3   = VMSize{CoreCount: 64, Family: standardESv3}
	VMSizeStandardE64iV3    = VMSize{CoreCount: 64, Family: standardESv3}
	VMSizeStandardE80isV4   = VMSize{CoreCount: 80, Family: standardEISv4}
	VMSizeStandardE80idsV4  = VMSize{CoreCount: 80, Family: standardEIDSv4}
	VMSizeStandardE104iV5   = VMSize{CoreCount: 104, Family: standardEIv5}
	VMSizeStandardE104isV5  = VMSize{CoreCount: 104, Family: standardEISv5}
	VMSizeStandardE104idV5  = VMSize{CoreCount: 104, Family: standardEIDv5}
	VMSizeStandardE104idsV5 = VMSize{CoreCount: 104, Family: standardEIDSv5}

	VMSizeStandardF4sV2  = VMSize{CoreCount: 4, Family: standardFSv2}
	VMSizeStandardF8sV2  = VMSize{CoreCount: 8, Family: standardFSv2}
	VMSizeStandardF16sV2 = VMSize{CoreCount: 16, Family: standardFSv2}
	VMSizeStandardF32sV2 = VMSize{CoreCount: 32, Family: standardFSv2}
	VMSizeStandardF72sV2 = VMSize{CoreCount: 72, Family: standardFSv2}

	VMSizeStandardM128ms = VMSize{CoreCount: 128, Family: standardMS}
	VMSizeStandardG5     = VMSize{CoreCount: 32, Family: standardGFamily}
	VMSizeStandardGS5    = VMSize{CoreCount: 32, Family: standardGFamily}

	VMSizeStandardL4s    = VMSize{CoreCount: 4, Family: standardLSv2}
	VMSizeStandardL8s    = VMSize{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16s   = VMSize{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32s   = VMSize{CoreCount: 32, Family: standardLSv2}
	VMSizeStandardL8sV2  = VMSize{CoreCount: 8, Family: standardLSv2}
	VMSizeStandardL16sV2 = VMSize{CoreCount: 16, Family: standardLSv2}
	VMSizeStandardL32sV2 = VMSize{CoreCount: 32, Family: standardLSv2}
	VMSizeStandardL48sV2 = VMSize{CoreCount: 48, Family: standardLSv2}
	VMSizeStandardL64sV2 = VMSize{CoreCount: 64, Family: standardLSv2}

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	VMSizeStandardNC4asT4V3  = VMSize{CoreCount: 4, Family: standardNCAS}
	VMSizeStandardNC8asT4V3  = VMSize{CoreCount: 8, Family: standardNCAS}
	VMSizeStandardNC16asT4V3 = VMSize{CoreCount: 16, Family: standardNCAS}
	VMSizeStandardNC64asT4V3 = VMSize{CoreCount: 64, Family: standardNCAS}
)

var supportedMasterVmSizes = map[api.VMSize]VMSize{
	// General purpose
	api.VMSizeStandardD8asV4:  VMSizeStandardD8asV4,
	api.VMSizeStandardD16asV4: VMSizeStandardD16asV4,
	api.VMSizeStandardD32asV4: VMSizeStandardD32asV4,

	// Memory optimized
	api.VMSizeStandardE64isV3:   VMSizeStandardE64isV3,
	api.VMSizeStandardE64iV3:    VMSizeStandardE64iV3,
	api.VMSizeStandardE80isV4:   VMSizeStandardE80isV4,
	api.VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4,
	api.VMSizeStandardE104iV5:   VMSizeStandardE104iV5,
	api.VMSizeStandardE104isV5:  VMSizeStandardE104isV5,
	api.VMSizeStandardE104idV5:  VMSizeStandardE104idV5,
	api.VMSizeStandardE104idsV5: VMSizeStandardE104idsV5,

	// Compute optimized
	api.VMSizeStandardF72sV2: VMSizeStandardF72sV2,

	// Memory and storage optimized
	api.VMSizeStandardG5:  VMSizeStandardG5,
	api.VMSizeStandardGS5: VMSizeStandardGS5,

	// Memory and compute optimized
	api.VMSizeStandardM128ms: VMSizeStandardM128ms,
}

var supportedWorkerVmSizes = map[api.VMSize]VMSize{
	api.VMSizeStandardD4asV4:  VMSizeStandardD4asV4,
	api.VMSizeStandardD8asV4:  VMSizeStandardD8asV4,
	api.VMSizeStandardD16asV4: VMSizeStandardD16asV4,
	api.VMSizeStandardD32asV4: VMSizeStandardD32asV4,

	api.VMSizeStandardD4sV3:  VMSizeStandardD4sV3,
	api.VMSizeStandardD8sV3:  VMSizeStandardD8sV3,
	api.VMSizeStandardD16sV3: VMSizeStandardD16sV3,
	api.VMSizeStandardD32sV3: VMSizeStandardD32sV3,

	api.VMSizeStandardE4sV3:     VMSizeStandardE4sV3,
	api.VMSizeStandardE8sV3:     VMSizeStandardE8sV3,
	api.VMSizeStandardE16sV3:    VMSizeStandardE16sV3,
	api.VMSizeStandardE32sV3:    VMSizeStandardE32sV3,
	api.VMSizeStandardE64isV3:   VMSizeStandardE64isV3,
	api.VMSizeStandardE64iV3:    VMSizeStandardE64iV3,
	api.VMSizeStandardE80isV4:   VMSizeStandardE80isV4,
	api.VMSizeStandardE80idsV4:  VMSizeStandardE80idsV4,
	api.VMSizeStandardE104iV5:   VMSizeStandardE104iV5,
	api.VMSizeStandardE104isV5:  VMSizeStandardE104isV5,
	api.VMSizeStandardE104idV5:  VMSizeStandardE104idV5,
	api.VMSizeStandardE104idsV5: VMSizeStandardE104idsV5,

	api.VMSizeStandardF4sV2:  VMSizeStandardF4sV2,
	api.VMSizeStandardF8sV2:  VMSizeStandardF8sV2,
	api.VMSizeStandardF16sV2: VMSizeStandardF16sV2,
	api.VMSizeStandardF32sV2: VMSizeStandardF32sV2,
	api.VMSizeStandardF72sV2: VMSizeStandardF72sV2,

	api.VMSizeStandardM128ms: VMSizeStandardM128ms,
	api.VMSizeStandardG5:     VMSizeStandardG5,
	api.VMSizeStandardGS5:    VMSizeStandardGS5,

	api.VMSizeStandardL4s:    VMSizeStandardL4s,
	api.VMSizeStandardL8s:    VMSizeStandardL8s,
	api.VMSizeStandardL16s:   VMSizeStandardL16s,
	api.VMSizeStandardL32s:   VMSizeStandardL32s,
	api.VMSizeStandardL8sV2:  VMSizeStandardL8sV2,
	api.VMSizeStandardL16sV2: VMSizeStandardL16sV2,
	api.VMSizeStandardL32sV2: VMSizeStandardL32sV2,
	api.VMSizeStandardL48sV2: VMSizeStandardL48sV2,
	api.VMSizeStandardL64sV2: VMSizeStandardL64sV2,

	// GPU nodes
	// the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// az vm list-usage -l eastus
	api.VMSizeStandardNC4asT4V3:  VMSizeStandardNC4asT4V3,
	api.VMSizeStandardNC8asT4V3:  VMSizeStandardNC8asT4V3,
	api.VMSizeStandardNC16asT4V3: VMSizeStandardNC16asT4V3,
	api.VMSizeStandardNC64asT4V3: VMSizeStandardNC64asT4V3,
}

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, requiredD2sV3Workers, isMaster bool) bool {
	if isMaster {
		_, supportedAsMaster := supportedMasterVmSizes[vmSize]
		return supportedAsMaster
	}

	_, supportedAsWorker := supportedWorkerVmSizes[vmSize]
	if supportedAsWorker || (requiredD2sV3Workers && vmSize == api.VMSizeStandardD2sV3) {
		return true
	}

	return false
}
