package vms

import (
	"github.com/Azure/ARO-RP/pkg/api/util/version"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// VMSize represents a VM size
type VMRole string

const (
	VMRoleMaster VMRole = "master"
	VMRoleWorker VMRole = "worker"
)

// VMSize represents a VM size
type VMSize string

func (vmSize VMSize) String() string {
	return string(vmSize)
}

// VMSize constants
const (
	VMSizeStandardD2sV3  VMSize = "Standard_D2s_v3"
	VMSizeStandardD4sV3  VMSize = "Standard_D4s_v3"
	VMSizeStandardD8sV3  VMSize = "Standard_D8s_v3"
	VMSizeStandardD16sV3 VMSize = "Standard_D16s_v3"
	VMSizeStandardD32sV3 VMSize = "Standard_D32s_v3"

	VMSizeStandardD2sV4  VMSize = "Standard_D2s_v4"
	VMSizeStandardD4sV4  VMSize = "Standard_D4s_v4"
	VMSizeStandardD8sV4  VMSize = "Standard_D8s_v4"
	VMSizeStandardD16sV4 VMSize = "Standard_D16s_v4"
	VMSizeStandardD32sV4 VMSize = "Standard_D32s_v4"
	VMSizeStandardD64sV4 VMSize = "Standard_D64s_v4"

	VMSizeStandardD2sV5  VMSize = "Standard_D2s_v5"
	VMSizeStandardD4sV5  VMSize = "Standard_D4s_v5"
	VMSizeStandardD8sV5  VMSize = "Standard_D8s_v5"
	VMSizeStandardD16sV5 VMSize = "Standard_D16s_v5"
	VMSizeStandardD32sV5 VMSize = "Standard_D32s_v5"
	VMSizeStandardD64sV5 VMSize = "Standard_D64s_v5"
	VMSizeStandardD96sV5 VMSize = "Standard_D96s_v5"

	VMSizeStandardD4asV4  VMSize = "Standard_D4as_v4"
	VMSizeStandardD8asV4  VMSize = "Standard_D8as_v4"
	VMSizeStandardD16asV4 VMSize = "Standard_D16as_v4"
	VMSizeStandardD32asV4 VMSize = "Standard_D32as_v4"
	VMSizeStandardD64asV4 VMSize = "Standard_D64as_v4"
	VMSizeStandardD96asV4 VMSize = "Standard_D96as_v4"

	VMSizeStandardD4asV5  VMSize = "Standard_D4as_v5"
	VMSizeStandardD8asV5  VMSize = "Standard_D8as_v5"
	VMSizeStandardD16asV5 VMSize = "Standard_D16as_v5"
	VMSizeStandardD32asV5 VMSize = "Standard_D32as_v5"
	VMSizeStandardD64asV5 VMSize = "Standard_D64as_v5"
	VMSizeStandardD96asV5 VMSize = "Standard_D96as_v5"

	VMSizeStandardD4dsV5  VMSize = "Standard_D4ds_v5"
	VMSizeStandardD8dsV5  VMSize = "Standard_D8ds_v5"
	VMSizeStandardD16dsV5 VMSize = "Standard_D16ds_v5"
	VMSizeStandardD32dsV5 VMSize = "Standard_D32ds_v5"
	VMSizeStandardD64dsV5 VMSize = "Standard_D64ds_v5"
	VMSizeStandardD96dsV5 VMSize = "Standard_D96ds_v5"

	VMSizeStandardD2sV6  VMSize = "Standard_D2s_v6"
	VMSizeStandardD4sV6  VMSize = "Standard_D4s_v6"
	VMSizeStandardD8sV6  VMSize = "Standard_D8s_v6"
	VMSizeStandardD16sV6 VMSize = "Standard_D16s_v6"
	VMSizeStandardD32sV6 VMSize = "Standard_D32s_v6"
	VMSizeStandardD48sV6 VMSize = "Standard_D48s_v6"
	VMSizeStandardD64sV6 VMSize = "Standard_D64s_v6"
	VMSizeStandardD96sV6 VMSize = "Standard_D96s_v6"

	VMSizeStandardD4dsV6  VMSize = "Standard_D4ds_v6"
	VMSizeStandardD8dsV6  VMSize = "Standard_D8ds_v6"
	VMSizeStandardD16dsV6 VMSize = "Standard_D16ds_v6"
	VMSizeStandardD32dsV6 VMSize = "Standard_D32ds_v6"
	VMSizeStandardD48dsV6 VMSize = "Standard_D48ds_v6"
	VMSizeStandardD64dsV6 VMSize = "Standard_D64ds_v6"
	VMSizeStandardD96dsV6 VMSize = "Standard_D96ds_v6"

	VMSizeStandardE4sV3  VMSize = "Standard_E4s_v3"
	VMSizeStandardE8sV3  VMSize = "Standard_E8s_v3"
	VMSizeStandardE16sV3 VMSize = "Standard_E16s_v3"
	VMSizeStandardE32sV3 VMSize = "Standard_E32s_v3"

	VMSizeStandardE2sV4  VMSize = "Standard_E2s_v4"
	VMSizeStandardE4sV4  VMSize = "Standard_E4s_v4"
	VMSizeStandardE8sV4  VMSize = "Standard_E8s_v4"
	VMSizeStandardE16sV4 VMSize = "Standard_E16s_v4"
	VMSizeStandardE20sV4 VMSize = "Standard_E20s_v4"
	VMSizeStandardE32sV4 VMSize = "Standard_E32s_v4"
	VMSizeStandardE48sV4 VMSize = "Standard_E48s_v4"
	VMSizeStandardE64sV4 VMSize = "Standard_E64s_v4"

	VMSizeStandardE2sV5  VMSize = "Standard_E2s_v5"
	VMSizeStandardE4sV5  VMSize = "Standard_E4s_v5"
	VMSizeStandardE8sV5  VMSize = "Standard_E8s_v5"
	VMSizeStandardE16sV5 VMSize = "Standard_E16s_v5"
	VMSizeStandardE20sV5 VMSize = "Standard_E20s_v5"
	VMSizeStandardE32sV5 VMSize = "Standard_E32s_v5"
	VMSizeStandardE48sV5 VMSize = "Standard_E48s_v5"
	VMSizeStandardE64sV5 VMSize = "Standard_E64s_v5"
	VMSizeStandardE96sV5 VMSize = "Standard_E96s_v5"

	VMSizeStandardE4asV4  VMSize = "Standard_E4as_v4"
	VMSizeStandardE8asV4  VMSize = "Standard_E8as_v4"
	VMSizeStandardE16asV4 VMSize = "Standard_E16as_v4"
	VMSizeStandardE20asV4 VMSize = "Standard_E20as_v4"
	VMSizeStandardE32asV4 VMSize = "Standard_E32as_v4"
	VMSizeStandardE48asV4 VMSize = "Standard_E48as_v4"
	VMSizeStandardE64asV4 VMSize = "Standard_E64as_v4"
	VMSizeStandardE96asV4 VMSize = "Standard_E96as_v4"

	VMSizeStandardE8asV5  VMSize = "Standard_E8as_v5"
	VMSizeStandardE16asV5 VMSize = "Standard_E16as_v5"
	VMSizeStandardE20asV5 VMSize = "Standard_E20as_v5"
	VMSizeStandardE32asV5 VMSize = "Standard_E32as_v5"
	VMSizeStandardE48asV5 VMSize = "Standard_E48as_v5"
	VMSizeStandardE64asV5 VMSize = "Standard_E64as_v5"
	VMSizeStandardE96asV5 VMSize = "Standard_E96as_v5"

	VMSizeStandardE64isV3   VMSize = "Standard_E64is_v3"
	VMSizeStandardE80isV4   VMSize = "Standard_E80is_v4"
	VMSizeStandardE80idsV4  VMSize = "Standard_E80ids_v4"
	VMSizeStandardE96dsV5   VMSize = "Standard_E96ds_v5"
	VMSizeStandardE104isV5  VMSize = "Standard_E104is_v5"
	VMSizeStandardE104idsV5 VMSize = "Standard_E104ids_v5"

	VMSizeStandardF4sV2  VMSize = "Standard_F4s_v2"
	VMSizeStandardF8sV2  VMSize = "Standard_F8s_v2"
	VMSizeStandardF16sV2 VMSize = "Standard_F16s_v2"
	VMSizeStandardF32sV2 VMSize = "Standard_F32s_v2"
	VMSizeStandardF72sV2 VMSize = "Standard_F72s_v2"

	VMSizeStandardM128ms VMSize = "Standard_M128ms"

	VMSizeStandardL4s  VMSize = "Standard_L4s"
	VMSizeStandardL8s  VMSize = "Standard_L8s"
	VMSizeStandardL16s VMSize = "Standard_L16s"
	VMSizeStandardL32s VMSize = "Standard_L32s"

	VMSizeStandardL8sV2  VMSize = "Standard_L8s_v2"
	VMSizeStandardL16sV2 VMSize = "Standard_L16s_v2"
	VMSizeStandardL32sV2 VMSize = "Standard_L32s_v2"
	VMSizeStandardL48sV2 VMSize = "Standard_L48s_v2"
	VMSizeStandardL64sV2 VMSize = "Standard_L64s_v2"

	VMSizeStandardL8sV3  VMSize = "Standard_L8s_v3"
	VMSizeStandardL16sV3 VMSize = "Standard_L16s_v3"
	VMSizeStandardL32sV3 VMSize = "Standard_L32s_v3"
	VMSizeStandardL48sV3 VMSize = "Standard_L48s_v3"
	VMSizeStandardL64sV3 VMSize = "Standard_L64s_v3"

	VMSizeStandardL4sV4  VMSize = "Standard_L4s_v4"
	VMSizeStandardL8sV4  VMSize = "Standard_L8s_v4"
	VMSizeStandardL16sV4 VMSize = "Standard_L16s_v4"
	VMSizeStandardL32sV4 VMSize = "Standard_L32s_v4"
	VMSizeStandardL48sV4 VMSize = "Standard_L48s_v4"
	VMSizeStandardL64sV4 VMSize = "Standard_L64s_v4"
	VMSizeStandardL80sV4 VMSize = "Standard_L80s_v4"

	VMSizeStandardD4lsV6  VMSize = "Standard_D4ls_v6"
	VMSizeStandardD8lsV6  VMSize = "Standard_D8ls_v6"
	VMSizeStandardD16lsV6 VMSize = "Standard_D16ls_v6"
	VMSizeStandardD32lsV6 VMSize = "Standard_D32ls_v6"
	VMSizeStandardD48lsV6 VMSize = "Standard_D48ls_v6"
	VMSizeStandardD64lsV6 VMSize = "Standard_D64ls_v6"
	VMSizeStandardD96lsV6 VMSize = "Standard_D96ls_v6"

	VMSizeStandardD4ldsV6  VMSize = "Standard_D4lds_v6"
	VMSizeStandardD8ldsV6  VMSize = "Standard_D8lds_v6"
	VMSizeStandardD16ldsV6 VMSize = "Standard_D1l6ds_v6"
	VMSizeStandardD32ldsV6 VMSize = "Standard_D32lds_v6"
	VMSizeStandardD48ldsV6 VMSize = "Standard_D48lds_v6"
	VMSizeStandardD64ldsV6 VMSize = "Standard_D64lds_v6"
	VMSizeStandardD96ldsV6 VMSize = "Standard_D96lds_v6"

	// GPU VMs
	VMSizeStandardNC4asT4V3  VMSize = "Standard_NC4as_T4_v3"
	VMSizeStandardNC8asT4V3  VMSize = "Standard_NC8as_T4_v3"
	VMSizeStandardNC16asT4V3 VMSize = "Standard_NC16as_T4_v3"
	VMSizeStandardNC64asT4V3 VMSize = "Standard_NC64as_T4_v3"

	VMSizeStandardNC6sV3   VMSize = "Standard_NC6s_v3"
	VMSizeStandardNC12sV3  VMSize = "Standard_NC12s_v3"
	VMSizeStandardNC24sV3  VMSize = "Standard_NC24s_v3"
	VMSizeStandardNC24rsV3 VMSize = "Standard_NC24rs_v3"
)

var ver419 = version.NewVersion(4, 19, 0)

// TODO: MAITIU - Fix JSON coding/decoding
type VMSizeStruct struct {
	CoreCount      int      // `json:"coreCount,omitempty"`
	Family         VMFamily //`json:"family,omitempty"`
	MinimumVersion version.Version
}

var (
	vmSizeStandardD2sV3Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv3}
	vmSizeStandardD4sV3Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv3}
	vmSizeStandardD8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv3}
	vmSizeStandardD16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv3}
	vmSizeStandardD32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv3}

	vmSizeStandardD2sV4Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv4}
	vmSizeStandardD4sV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv4}
	vmSizeStandardD8sV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv4}
	vmSizeStandardD16sV4Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv4}
	vmSizeStandardD32sV4Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv4}
	vmSizeStandardD64sV4Struct = VMSizeStruct{CoreCount: 64, Family: standardDSv4}

	vmSizeStandardD2sV5Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv5}
	vmSizeStandardD4sV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv5}
	vmSizeStandardD8sV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv5}
	vmSizeStandardD16sV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv5}
	vmSizeStandardD32sV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv5}
	vmSizeStandardD64sV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDSv5}
	vmSizeStandardD96sV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDSv5}

	vmSizeStandardD4asV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardDASv4}
	vmSizeStandardD8asV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardDASv4}
	vmSizeStandardD16asV4Struct = VMSizeStruct{CoreCount: 16, Family: standardDASv4}
	vmSizeStandardD32asV4Struct = VMSizeStruct{CoreCount: 32, Family: standardDASv4}
	vmSizeStandardD64asV4Struct = VMSizeStruct{CoreCount: 64, Family: standardDASv4}
	vmSizeStandardD96asV4Struct = VMSizeStruct{CoreCount: 96, Family: standardDASv4}

	vmSizeStandardD4asV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDASv5}
	vmSizeStandardD8asV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDASv5}
	vmSizeStandardD16asV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDASv5}
	vmSizeStandardD32asV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDASv5}
	vmSizeStandardD64asV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDASv5}
	vmSizeStandardD96asV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDASv5}

	vmSizeStandardD4dsV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardDDSv5}
	vmSizeStandardD8dsV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardDDSv5}
	vmSizeStandardD16dsV5Struct = VMSizeStruct{CoreCount: 16, Family: standardDDSv5}
	vmSizeStandardD32dsV5Struct = VMSizeStruct{CoreCount: 32, Family: standardDDSv5}
	vmSizeStandardD64dsV5Struct = VMSizeStruct{CoreCount: 64, Family: standardDDSv5}
	vmSizeStandardD96dsV5Struct = VMSizeStruct{CoreCount: 96, Family: standardDDSv5}

	vmSizeStandardD2sV6Struct  = VMSizeStruct{CoreCount: 2, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD4sV6Struct  = VMSizeStruct{CoreCount: 4, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD8sV6Struct  = VMSizeStruct{CoreCount: 8, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD16sV6Struct = VMSizeStruct{CoreCount: 16, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD32sV6Struct = VMSizeStruct{CoreCount: 32, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD48sV6Struct = VMSizeStruct{CoreCount: 48, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD64sV6Struct = VMSizeStruct{CoreCount: 64, Family: standardDSv6, MinimumVersion: ver419}
	vmSizeStandardD96sV6Struct = VMSizeStruct{CoreCount: 96, Family: standardDSv6, MinimumVersion: ver419}

	vmSizeStandardD4dsV6Struct  = VMSizeStruct{CoreCount: 4, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD8dsV6Struct  = VMSizeStruct{CoreCount: 8, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD16dsV6Struct = VMSizeStruct{CoreCount: 16, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD32dsV6Struct = VMSizeStruct{CoreCount: 32, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD48dsV6Struct = VMSizeStruct{CoreCount: 48, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD64dsV6Struct = VMSizeStruct{CoreCount: 64, Family: standardDDSv6, MinimumVersion: ver419}
	vmSizeStandardD96dsV6Struct = VMSizeStruct{CoreCount: 96, Family: standardDDSv6, MinimumVersion: ver419}

	vmSizeStandardE4sV3Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv3}
	vmSizeStandardE8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv3}
	vmSizeStandardE16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardESv3}
	vmSizeStandardE32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardESv3}

	vmSizeStandardE2sV4Struct  = VMSizeStruct{CoreCount: 2, Family: standardESv4}
	vmSizeStandardE4sV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv4}
	vmSizeStandardE8sV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv4}
	vmSizeStandardE16sV4Struct = VMSizeStruct{CoreCount: 16, Family: standardESv4}
	vmSizeStandardE20sV4Struct = VMSizeStruct{CoreCount: 20, Family: standardESv4}
	vmSizeStandardE32sV4Struct = VMSizeStruct{CoreCount: 32, Family: standardESv4}
	vmSizeStandardE48sV4Struct = VMSizeStruct{CoreCount: 48, Family: standardESv4}
	vmSizeStandardE64sV4Struct = VMSizeStruct{CoreCount: 64, Family: standardESv4}

	vmSizeStandardE2sV5Struct  = VMSizeStruct{CoreCount: 2, Family: standardESv5}
	vmSizeStandardE4sV5Struct  = VMSizeStruct{CoreCount: 4, Family: standardESv5}
	vmSizeStandardE8sV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardESv5}
	vmSizeStandardE16sV5Struct = VMSizeStruct{CoreCount: 16, Family: standardESv5}
	vmSizeStandardE20sV5Struct = VMSizeStruct{CoreCount: 20, Family: standardESv5}
	vmSizeStandardE32sV5Struct = VMSizeStruct{CoreCount: 32, Family: standardESv5}
	vmSizeStandardE48sV5Struct = VMSizeStruct{CoreCount: 48, Family: standardESv5}
	vmSizeStandardE64sV5Struct = VMSizeStruct{CoreCount: 64, Family: standardESv5}
	vmSizeStandardE96sV5Struct = VMSizeStruct{CoreCount: 96, Family: standardESv5}

	vmSizeStandardE4asV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardEASv4}
	vmSizeStandardE8asV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardEASv4}
	vmSizeStandardE16asV4Struct = VMSizeStruct{CoreCount: 16, Family: standardEASv4}
	vmSizeStandardE20asV4Struct = VMSizeStruct{CoreCount: 20, Family: standardEASv4}
	vmSizeStandardE32asV4Struct = VMSizeStruct{CoreCount: 32, Family: standardEASv4}
	vmSizeStandardE48asV4Struct = VMSizeStruct{CoreCount: 48, Family: standardEASv4}
	vmSizeStandardE64asV4Struct = VMSizeStruct{CoreCount: 64, Family: standardEASv4}
	vmSizeStandardE96asV4Struct = VMSizeStruct{CoreCount: 96, Family: standardEASv4}

	vmSizeStandardE8asV5Struct  = VMSizeStruct{CoreCount: 8, Family: standardEASv5}
	vmSizeStandardE16asV5Struct = VMSizeStruct{CoreCount: 16, Family: standardEASv5}
	vmSizeStandardE20asV5Struct = VMSizeStruct{CoreCount: 20, Family: standardEASv5}
	vmSizeStandardE32asV5Struct = VMSizeStruct{CoreCount: 32, Family: standardEASv5}
	vmSizeStandardE48asV5Struct = VMSizeStruct{CoreCount: 48, Family: standardEASv5}
	vmSizeStandardE64asV5Struct = VMSizeStruct{CoreCount: 64, Family: standardEASv5}
	vmSizeStandardE96asV5Struct = VMSizeStruct{CoreCount: 96, Family: standardEASv5}

	vmSizeStandardE64isV3Struct   = VMSizeStruct{CoreCount: 64, Family: standardESv3}
	vmSizeStandardE80isV4Struct   = VMSizeStruct{CoreCount: 80, Family: standardEISv4}
	vmSizeStandardE80idsV4Struct  = VMSizeStruct{CoreCount: 80, Family: standardEIDSv4}
	vmSizeStandardE96dsV5Struct   = VMSizeStruct{CoreCount: 96, Family: standardEDSv5}
	vmSizeStandardE104isV5Struct  = VMSizeStruct{CoreCount: 104, Family: standardEISv5}
	vmSizeStandardE104idsV5Struct = VMSizeStruct{CoreCount: 104, Family: standardEIDSv5}

	vmSizeStandardF4sV2Struct  = VMSizeStruct{CoreCount: 4, Family: standardFSv2}
	vmSizeStandardF8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardFSv2}
	vmSizeStandardF16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardFSv2}
	vmSizeStandardF32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardFSv2}
	vmSizeStandardF72sV2Struct = VMSizeStruct{CoreCount: 72, Family: standardFSv2}

	vmSizeStandardM128msStruct = VMSizeStruct{CoreCount: 128, Family: standardMS}

	vmSizeStandardL4sStruct  = VMSizeStruct{CoreCount: 4, Family: standardLSv2}
	vmSizeStandardL8sStruct  = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	vmSizeStandardL16sStruct = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	vmSizeStandardL32sStruct = VMSizeStruct{CoreCount: 32, Family: standardLSv2}

	vmSizeStandardL8sV2Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv2}
	vmSizeStandardL16sV2Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv2}
	vmSizeStandardL32sV2Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv2}
	vmSizeStandardL48sV2Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv2}
	vmSizeStandardL64sV2Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv2}

	vmSizeStandardL8sV3Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv3}
	vmSizeStandardL16sV3Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv3}
	vmSizeStandardL32sV3Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv3}
	vmSizeStandardL48sV3Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv3}
	vmSizeStandardL64sV3Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv3}

	vmSizeStandardL4sV4Struct  = VMSizeStruct{CoreCount: 4, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL8sV4Struct  = VMSizeStruct{CoreCount: 8, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL16sV4Struct = VMSizeStruct{CoreCount: 16, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL32sV4Struct = VMSizeStruct{CoreCount: 32, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL48sV4Struct = VMSizeStruct{CoreCount: 48, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL64sV4Struct = VMSizeStruct{CoreCount: 64, Family: standardLSv4, MinimumVersion: ver419}
	vmSizeStandardL80sV4Struct = VMSizeStruct{CoreCount: 80, Family: standardLSv4, MinimumVersion: ver419}

	vmSizeStandardD4lsV6Struct  = VMSizeStruct{CoreCount: 4, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD8lsV6Struct  = VMSizeStruct{CoreCount: 8, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD16lsV6Struct = VMSizeStruct{CoreCount: 16, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD32lsV6Struct = VMSizeStruct{CoreCount: 32, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD48lsV6Struct = VMSizeStruct{CoreCount: 48, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD64lsV6Struct = VMSizeStruct{CoreCount: 64, Family: standardDLSv6, MinimumVersion: ver419}
	vmSizeStandardD96lsV6Struct = VMSizeStruct{CoreCount: 96, Family: standardDLSv6, MinimumVersion: ver419}

	vmSizeStandardD4ldsV6Struct  = VMSizeStruct{CoreCount: 4, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD8ldsV6Struct  = VMSizeStruct{CoreCount: 8, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD16ldsV6Struct = VMSizeStruct{CoreCount: 16, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD32ldsV6Struct = VMSizeStruct{CoreCount: 32, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD48ldsV6Struct = VMSizeStruct{CoreCount: 48, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD64ldsV6Struct = VMSizeStruct{CoreCount: 64, Family: standardDLDSv6, MinimumVersion: ver419}
	vmSizeStandardD96ldsV6Struct = VMSizeStruct{CoreCount: 96, Family: standardDLDSv6, MinimumVersion: ver419}

	// Struct GPU nodes
	// Struct the formatting of the ncasv3_t4 family is different.  This can be seen through a
	// Struct az vm list-usage -l eastus
	vmSizeStandardNC4asT4V3Struct  = VMSizeStruct{CoreCount: 4, Family: standardNCAS}
	vmSizeStandardNC8asT4V3Struct  = VMSizeStruct{CoreCount: 8, Family: standardNCAS}
	vmSizeStandardNC16asT4V3Struct = VMSizeStruct{CoreCount: 16, Family: standardNCAS}
	vmSizeStandardNC64asT4V3Struct = VMSizeStruct{CoreCount: 64, Family: standardNCAS}

	vmSizeStandardNC6sV3Struct   = VMSizeStruct{CoreCount: 6, Family: standardNCSv3}
	vmSizeStandardNC12sV3Struct  = VMSizeStruct{CoreCount: 12, Family: standardNCSv3}
	vmSizeStandardNC24sV3Struct  = VMSizeStruct{CoreCount: 24, Family: standardNCSv3}
	vmSizeStandardNC24rsV3Struct = VMSizeStruct{CoreCount: 24, Family: standardNCSv3}
)

// VMFamily represents a VM family
type VMFamily string

func (vmFamily VMFamily) String() string {
	return string(vmFamily)
}

const (
	standardDSv3   VMFamily = "standardDSv3Family"
	standardDSv4   VMFamily = "standardDSv4Family"
	standardDSv5   VMFamily = "standardDSv5Family"
	standardDSv6   VMFamily = "standardDSv6Family"
	standardDASv4  VMFamily = "standardDASv4Family"
	standardDASv5  VMFamily = "standardDASv5Family"
	standardDDSv5  VMFamily = "standardDDSv5Family"
	standardDDSv6  VMFamily = "standardDDSv6Family"
	standardESv3   VMFamily = "standardESv3Family"
	standardESv4   VMFamily = "standardESv4Family"
	standardESv5   VMFamily = "standardESv5Family"
	standardEASv4  VMFamily = "standardEASv4Family"
	standardEASv5  VMFamily = "standardEASv5Family"
	standardEISv4  VMFamily = "standardEISv4Family"
	standardEIDSv4 VMFamily = "standardEIDSv4Family"
	standardEISv5  VMFamily = "standardEISv5Family"
	standardEDSv5  VMFamily = "standardEDSv5Family"
	standardEIDSv5 VMFamily = "standardEIDSv5Family"
	standardEIDv5  VMFamily = "standardEIDv5Family"
	standardFSv2   VMFamily = "standardFSv2Family"
	standardMS     VMFamily = "standardMSFamily"
	standardLSv2   VMFamily = "standardLsv2Family"
	standardLSv3   VMFamily = "standardLsv3Family"
	standardLSv4   VMFamily = "standardLsv4Family"
	standardDLSv6  VMFamily = "standardDLSv6Family"
	standardDLDSv6 VMFamily = "standardDLDSv6Family"
	standardNCAS   VMFamily = "Standard NCASv3_T4 Family"
	standardNCSv3  VMFamily = "Standard NCSv3 Family"
)
