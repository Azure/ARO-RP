package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, requiredD2sV3Workers, isMaster bool) bool {
	if isMaster {
		switch vmSize {
		case api.VMSizeStandardD8sV3,
			api.VMSizeStandardD16sV3,
			api.VMSizeStandardD32sV3,
			api.VMSizeStandardE64iV3,
			api.VMSizeStandardE64isV3,
			api.VMSizeStandardF72sV2,
			api.VMSizeStandardGS5,
			api.VMSizeStandardG5,
			api.VMSizeStandardM128ms:
			return true
		}
	} else {
		if requiredD2sV3Workers {
			switch vmSize {
			case api.VMSizeStandardD2sV3:
				return true
			}
		} else {
			switch vmSize {
			case api.VMSizeStandardD4asV4,
				api.VMSizeStandardD8asV4,
				api.VMSizeStandardD16asV4,
				api.VMSizeStandardD32asV4,
				api.VMSizeStandardD4sV3,
				api.VMSizeStandardD8sV3,
				api.VMSizeStandardD16sV3,
				api.VMSizeStandardD32sV3,
				api.VMSizeStandardE4sV3,
				api.VMSizeStandardE8sV3,
				api.VMSizeStandardE16sV3,
				api.VMSizeStandardE32sV3,
				api.VMSizeStandardE64iV3,
				api.VMSizeStandardE64isV3,
				api.VMSizeStandardF4sV2,
				api.VMSizeStandardF8sV2,
				api.VMSizeStandardF16sV2,
				api.VMSizeStandardF32sV2,
				api.VMSizeStandardF72sV2,
				api.VMSizeStandardG5,
				api.VMSizeStandardGS5,
				api.VMSizeStandardM128ms,
				api.VMSizeStandardL4s,
				api.VMSizeStandardL8s,
				api.VMSizeStandardL16s,
				api.VMSizeStandardL32s,
				api.VMSizeStandardL8sV2,
				api.VMSizeStandardL16sV2,
				api.VMSizeStandardL32sV2,
				api.VMSizeStandardL48sV2,
				api.VMSizeStandardL64sV2:
				return true
			}
		}
	}
	return false
}
