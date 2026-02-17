package vms

import "github.com/Azure/ARO-RP/pkg/api"
import "maps"


var SupportedMasterVmSizes = map[api.VMSize]VMSizeStruct{
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

	api.VMSizeStandardD8dsV5:  api.VMSizeStandardD8dsV5Struct,
	api.VMSizeStandardD16dsV5: api.VMSizeStandardD16dsV5Struct,
	api.VMSizeStandardD32dsV5: api.VMSizeStandardD32dsV5Struct,

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
	api.VMSizeStandardE96asV4: api.VMSizeStandardE96asV4Struct,

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

	api.VMSizeStandardD4sV6:  api.VMSizeStandardD4sV6Struct,
	api.VMSizeStandardD8sV6:  api.VMSizeStandardD8sV6Struct,
	api.VMSizeStandardD16sV6: api.VMSizeStandardD16sV6Struct,
	api.VMSizeStandardD32sV6: api.VMSizeStandardD32sV6Struct,
	api.VMSizeStandardD48sV6: api.VMSizeStandardD48sV6Struct,
	api.VMSizeStandardD64sV6: api.VMSizeStandardD64sV6Struct,
	api.VMSizeStandardD96sV6: api.VMSizeStandardD96sV6Struct,

	api.VMSizeStandardD4dsV6:  api.VMSizeStandardD4dsV6Struct,
	api.VMSizeStandardD8dsV6:  api.VMSizeStandardD8dsV6Struct,
	api.VMSizeStandardD16dsV6: api.VMSizeStandardD16dsV6Struct,
	api.VMSizeStandardD32dsV6: api.VMSizeStandardD32dsV6Struct,
	api.VMSizeStandardD48dsV6: api.VMSizeStandardD48dsV6Struct,
	api.VMSizeStandardD64dsV6: api.VMSizeStandardD64dsV6Struct,
	api.VMSizeStandardD96dsV6: api.VMSizeStandardD96dsV6Struct,
}

var SupportedWorkerVmSizes = map[api.VMSize]VMSizeStruct{
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

	api.VMSizeStandardD4dsV5:  api.VMSizeStandardD4dsV5Struct,
	api.VMSizeStandardD8dsV5:  api.VMSizeStandardD8dsV5Struct,
	api.VMSizeStandardD16dsV5: api.VMSizeStandardD16dsV5Struct,
	api.VMSizeStandardD32dsV5: api.VMSizeStandardD32dsV5Struct,
	api.VMSizeStandardD64dsV5: api.VMSizeStandardD64dsV5Struct,
	api.VMSizeStandardD96dsV5: api.VMSizeStandardD96dsV5Struct,

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
	api.VMSizeStandardE96asV4: api.VMSizeStandardE96asV4Struct,

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

	api.VMSizeStandardL4sV4:  api.VMSizeStandardL4sV4Struct,
	api.VMSizeStandardL8sV4:  api.VMSizeStandardL8sV4Struct,
	api.VMSizeStandardL16sV4: api.VMSizeStandardL16sV4Struct,
	api.VMSizeStandardL32sV4: api.VMSizeStandardL32sV4Struct,
	api.VMSizeStandardL48sV4: api.VMSizeStandardL48sV4Struct,
	api.VMSizeStandardL64sV4: api.VMSizeStandardL64sV4Struct,
	api.VMSizeStandardL80sV4: api.VMSizeStandardL80sV4Struct,

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

	api.VMSizeStandardD4sV6:  api.VMSizeStandardD4sV6Struct,
	api.VMSizeStandardD8sV6:  api.VMSizeStandardD8sV6Struct,
	api.VMSizeStandardD16sV6: api.VMSizeStandardD16sV6Struct,
	api.VMSizeStandardD32sV6: api.VMSizeStandardD32sV6Struct,
	api.VMSizeStandardD48sV6: api.VMSizeStandardD48sV6Struct,
	api.VMSizeStandardD64sV6: api.VMSizeStandardD64sV6Struct,
	api.VMSizeStandardD96sV6: api.VMSizeStandardD96sV6Struct,

	api.VMSizeStandardD4dsV6:  api.VMSizeStandardD4dsV6Struct,
	api.VMSizeStandardD8dsV6:  api.VMSizeStandardD8dsV6Struct,
	api.VMSizeStandardD16dsV6: api.VMSizeStandardD16dsV6Struct,
	api.VMSizeStandardD32dsV6: api.VMSizeStandardD32dsV6Struct,
	api.VMSizeStandardD48dsV6: api.VMSizeStandardD48dsV6Struct,
	api.VMSizeStandardD64dsV6: api.VMSizeStandardD64dsV6Struct,
	api.VMSizeStandardD96dsV6: api.VMSizeStandardD96dsV6Struct,

	api.VMSizeStandardD4lsV6:  api.VMSizeStandardD4lsV6Struct,
	api.VMSizeStandardD8lsV6:  api.VMSizeStandardD8lsV6Struct,
	api.VMSizeStandardD16lsV6: api.VMSizeStandardD16lsV6Struct,
	api.VMSizeStandardD32lsV6: api.VMSizeStandardD32lsV6Struct,
	api.VMSizeStandardD48lsV6: api.VMSizeStandardD48lsV6Struct,
	api.VMSizeStandardD64lsV6: api.VMSizeStandardD64lsV6Struct,
	api.VMSizeStandardD96lsV6: api.VMSizeStandardD96lsV6Struct,

	api.VMSizeStandardD4ldsV6:  api.VMSizeStandardD4ldsV6Struct,
	api.VMSizeStandardD8ldsV6:  api.VMSizeStandardD8ldsV6Struct,
	api.VMSizeStandardD16ldsV6: api.VMSizeStandardD16ldsV6Struct,
	api.VMSizeStandardD32ldsV6: api.VMSizeStandardD32ldsV6Struct,
	api.VMSizeStandardD48ldsV6: api.VMSizeStandardD48ldsV6Struct,
	api.VMSizeStandardD64ldsV6: api.VMSizeStandardD64ldsV6Struct,
	api.VMSizeStandardD96ldsV6: api.VMSizeStandardD96ldsV6Struct,
}

var SupportedMasterVmSizesForInternalUser = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD4sV3: api.VMSizeStandardD4sV3Struct,
	api.VMSizeStandardD4sV4: api.VMSizeStandardD4sV4Struct,
	api.VMSizeStandardD4sV5: api.VMSizeStandardD4sV5Struct,
	api.VMSizeStandardD4sV6: api.VMSizeStandardD4sV6Struct,
}

var SupportedWorkerVmSizesForInternalUser = map[api.VMSize]VMSizeStruct{
	api.VMSizeStandardD2sV3: api.VMSizeStandardD2sV3Struct,
	api.VMSizeStandardD2sV4: api.VMSizeStandardD2sV4Struct,
	api.VMSizeStandardD2sV5: api.VMSizeStandardD2sV5Struct,
	api.VMSizeStandardD2sV6: api.VMSizeStandardD2sV6Struct,
}

maps.Copy(SupportedMasterVmSizesForInternalUser, SupportedMasterVmSizes)
maps.Copy(SupportedWorkerVmSizesForInternalUser, SupportedWorkerVmSizes)
