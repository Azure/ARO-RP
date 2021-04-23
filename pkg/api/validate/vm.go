package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize api.VMSize, requireD2sV3Workers, isMaster bool) bool {
	if isMaster {
		switch vmSize {
		case api.VMSizeStandardD8sV3,
			api.VMSizeStandardD16sV3,
			api.VMSizeStandardD32sV3:
			return true
		}
	} else {
		if requireD2sV3Workers {
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
				api.VMSizeStandardF4sV2,
				api.VMSizeStandardF8sV2,
				api.VMSizeStandardF16sV2,
				api.VMSizeStandardF32sV2:
				return true
			}
		}
	}

	return false
}
