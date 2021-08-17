package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
)

func addRequiredResources(requiredResources map[string]int, vmSize api.VMSize, count int) error {
	requiredResources["virtualMachines"] += count
	requiredResources["PremiumDiskCount"] += count
	switch vmSize {
	case api.VMSizeStandardD2sV3:
		requiredResources["standardDSv3Family"] += (count * 2)
		requiredResources["cores"] += (count * 2)

	case api.VMSizeStandardD4asV4:
		requiredResources["standardDASv4Family"] += (count * 4)
		requiredResources["cores"] += (count * 4)
	case api.VMSizeStandardD8asV4:
		requiredResources["standardDASv4Family"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardD16asV4:
		requiredResources["standardDASv4Family"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardD32asV4:
		requiredResources["standardDASv4Family"] += (count * 32)
		requiredResources["cores"] += (count * 32)

	case api.VMSizeStandardD4sV3:
		requiredResources["standardDSv3Family"] += (count * 4)
		requiredResources["cores"] += (count * 4)
	case api.VMSizeStandardD8sV3:
		requiredResources["standardDSv3Family"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardD16sV3:
		requiredResources["standardDSv3Family"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardD32sV3:
		requiredResources["standardDSv3Family"] += (count * 32)
		requiredResources["cores"] += (count * 32)

	case api.VMSizeStandardE4sV3:
		requiredResources["standardESv3Family"] += (count * 4)
		requiredResources["cores"] += (count * 4)
	case api.VMSizeStandardE8sV3:
		requiredResources["standardESv3Family"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardE16sV3:
		requiredResources["standardESv3Family"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardE32sV3:
		requiredResources["standardESv3Family"] += (count * 32)
		requiredResources["cores"] += (count * 32)
		//Support for Compute isolation
	case api.VMSizeStandardE64iV3:
		requiredResources["standardEIv3Family"] += (count * 64)
		requiredResources["cores"] += (count * 64)
	case api.VMSizeStandardE64isV3:
		requiredResources["standardEISv3Family"] += (count * 64)
		requiredResources["cores"] += (count * 64)
	case api.VMSizeStandardF4sV2:
		requiredResources["standardFSv2Family"] += (count * 4)
		requiredResources["cores"] += (count * 4)
	case api.VMSizeStandardF8sV2:
		requiredResources["standardFSv2Family"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardF16sV2:
		requiredResources["standardFSv2Family"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardF32sV2:
		requiredResources["standardFSv2Family"] += (count * 32)
		requiredResources["cores"] += (count * 32)
	case api.VMSizeStandardF72sV2:
		requiredResources["standardFSv2Family"] += (count * 72)
		requiredResources["cores"] += (count * 72)
	case api.VMSizeStandardM128ms:
		requiredResources["standardMSFamily"] += (count * 128)
		requiredResources["cores"] += (count * 128)
	case api.VMSizeStandardG5:
		requiredResources["standardGFamily"] += (count * 32)
		requiredResources["cores"] += (count * 32)
	case api.VMSizeStandardGS5:
		requiredResources["standardGSFamily"] += (count * 32)
		requiredResources["cores"] += (count * 32)
	case api.VMSizeStandardL4s:
		requiredResources["standardLsFamily"] += (count * 4)
		requiredResources["cores"] += (count * 4)
	case api.VMSizeStandardL8s:
		requiredResources["standardLsFamily"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardL16s:
		requiredResources["standardLsFamily"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardL32s:
		requiredResources["standardLsFamily"] += (count * 32)
		requiredResources["cores"] += (count * 32)
	case api.VMSizeStandardL8sV2:
		requiredResources["standardLsv2Family"] += (count * 8)
		requiredResources["cores"] += (count * 8)
	case api.VMSizeStandardL16sV2:
		requiredResources["standardLsv2Family"] += (count * 16)
		requiredResources["cores"] += (count * 16)
	case api.VMSizeStandardL32sV2:
		requiredResources["standardLsv2Family"] += (count * 32)
		requiredResources["cores"] += (count * 32)
	case api.VMSizeStandardL48sV2:
		requiredResources["standardLsv2Family"] += (count * 48)
		requiredResources["cores"] += (count * 48)
	case api.VMSizeStandardL64sV2:
		requiredResources["standardLsv2Family"] += (count * 64)
		requiredResources["cores"] += (count * 64)
	default:
		//will only happen if pkg/api verification allows new VMSizes
		return fmt.Errorf("unexpected node VMSize %s", vmSize)
	}
	return nil
}

// ValidateQuota checks usage quotas vs. resources required by cluster before cluster
// creation
func (dv *dynamic) ValidateQuota(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("ValidateQuota")

	// If ValidateQuota runs outside install process, we should skip quota validation
	if oc.Properties.Install == nil || oc.Properties.Install.Phase != api.InstallPhaseBootstrap {
		return nil
	}

	requiredResources := map[string]int{}
	err := addRequiredResources(requiredResources, oc.Properties.MasterProfile.VMSize, 3)
	if err != nil {
		return err
	}
	//worker node resource calculation
	for _, w := range oc.Properties.WorkerProfiles {
		err = addRequiredResources(requiredResources, w.VMSize, w.Count)
		if err != nil {
			return err
		}
	}

	//Public IP Addresses minimum requirement: 2 for ARM template deployment and 1 for kube-controller-manager
	requiredResources["PublicIPAddresses"] = 3

	//check requirements vs. usage

	// we're only checking the limits returned by the Usage API and ignoring usage limits missing from the results
	// rationale:
	// 1. if the Usage API doesn't send a limit because a resource is no longer limited, RP will continue cluster creation without impact
	// 2. if the Usage API doesn't send a limit that is still enforced, cluster creation will fail on the backend and we will get an error in the RP logs
	computeUsages, err := dv.spComputeUsage.List(ctx, oc.Location)
	if err != nil {
		return err
	}

	for _, usage := range computeUsages {
		required, present := requiredResources[*usage.Name.Value]
		if present && int64(required) > (*usage.Limit-int64(*usage.CurrentValue)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "", "Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.", *usage.Name.Value, *usage.Limit, *usage.CurrentValue, required)
		}
	}

	netUsages, err := dv.spNetworkUsage.List(ctx, oc.Location)
	if err != nil {
		return err
	}

	for _, netUsage := range netUsages {
		required, present := requiredResources[*netUsage.Name.Value]
		if present && int64(required) > (*netUsage.Limit-int64(*netUsage.CurrentValue)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "", "Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.", *netUsage.Name.Value, *netUsage.Limit, *netUsage.CurrentValue, required)
		}
	}

	return nil
}
