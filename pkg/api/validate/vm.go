package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func DiskSizeIsValid(sizeGB int) bool {
	return sizeGB >= 128
}

func VMSizeIsValid(vmSize VMSize, requiredD2sV3Workers, isMaster bool) bool {
	if isMaster {
		switch vmSize {
		case VMSizeStandardD8sV3,
			VMSizeStandardD16sV3,
			VMSizeStandardD32sV3,
			VMSizeStandardE64iV3,
			VMSizeStandardE64isV3,
			VMSizeStandardF72sV2,
			VMSizeStandardGS5,
			VMSizeStandardG5,
			VMSizeStandardM128ms:
			return true
		}
	} else {
		if requiredD2sV3Workers {
			switch vmSize {
			case VMSizeStandardD2sV3:
				return true
			}
		} else {
			switch vmSize {
			case VMSizeStandardD4asV4,
				VMSizeStandardD8asV4,
				VMSizeStandardD16asV4,
				VMSizeStandardD32asV4,
				VMSizeStandardD4sV3,
				VMSizeStandardD8sV3,
				VMSizeStandardD16sV3,
				VMSizeStandardD32sV3,
				VMSizeStandardE4sV3,
				VMSizeStandardE8sV3,
				VMSizeStandardE16sV3,
				VMSizeStandardE32sV3,
				VMSizeStandardE64iV3,
				VMSizeStandardE64isV3,
				VMSizeStandardF4sV2,
				VMSizeStandardF8sV2,
				VMSizeStandardF16sV2,
				VMSizeStandardF32sV2,
				VMSizeStandardF72sV2,
				VMSizeStandardG5,
				VMSizeStandardGS5,
				VMSizeStandardM128ms:
				return true
			}
		}
	}
	return false
}
