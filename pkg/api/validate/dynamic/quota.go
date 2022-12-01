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
	vmTypesMap := map[api.VMSize]struct {
		CoreCount int
		Family    string
	}{
		api.VMSizeStandardD2sV3: {CoreCount: 2, Family: "standardDSv3Family"},

		api.VMSizeStandardD4asV4:  {CoreCount: 4, Family: "standardDASv4Family"},
		api.VMSizeStandardD8asV4:  {CoreCount: 8, Family: "standardDASv4Family"},
		api.VMSizeStandardD16asV4: {CoreCount: 16, Family: "standardDASv4Family"},
		api.VMSizeStandardD32asV4: {CoreCount: 32, Family: "standardDASv4Family"},

		api.VMSizeStandardD4sV3:  {CoreCount: 4, Family: "standardDSv3Family"},
		api.VMSizeStandardD8sV3:  {CoreCount: 8, Family: "standardDSv3Family"},
		api.VMSizeStandardD16sV3: {CoreCount: 16, Family: "standardDSv3Family"},
		api.VMSizeStandardD32sV3: {CoreCount: 32, Family: "standardDSv3Family"},

		api.VMSizeStandardE4sV3:     {CoreCount: 4, Family: "standardESv3Family"},
		api.VMSizeStandardE8sV3:     {CoreCount: 8, Family: "standardESv3Family"},
		api.VMSizeStandardE16sV3:    {CoreCount: 16, Family: "standardESv3Family"},
		api.VMSizeStandardE32sV3:    {CoreCount: 32, Family: "standardESv3Family"},
		api.VMSizeStandardE64isV3:   {CoreCount: 64, Family: "standardESv3Family"},
		api.VMSizeStandardE64iV3:    {CoreCount: 64, Family: "standardESv3Family"},
		api.VMSizeStandardE80isV4:   {CoreCount: 80, Family: "standardEISv4Family"},
		api.VMSizeStandardE80idsV4:  {CoreCount: 80, Family: "standardEIDSv4Family"},
		api.VMSizeStandardE104iV5:   {CoreCount: 104, Family: "standardEIv5Family"},
		api.VMSizeStandardE104isV5:  {CoreCount: 104, Family: "standardEISv5Family"},
		api.VMSizeStandardE104idV5:  {CoreCount: 104, Family: "standardEIDv5Family"},
		api.VMSizeStandardE104idsV5: {CoreCount: 104, Family: "standardEIDSv5Family"},

		api.VMSizeStandardF4sV2:  {CoreCount: 4, Family: "standardFSv2Family"},
		api.VMSizeStandardF8sV2:  {CoreCount: 8, Family: "standardFSv2Family"},
		api.VMSizeStandardF16sV2: {CoreCount: 16, Family: "standardFSv2Family"},
		api.VMSizeStandardF32sV2: {CoreCount: 32, Family: "standardFSv2Family"},
		api.VMSizeStandardF72sV2: {CoreCount: 72, Family: "standardFSv2Family"},

		api.VMSizeStandardM128ms: {CoreCount: 128, Family: "standardMSFamily"},
		api.VMSizeStandardG5:     {CoreCount: 32, Family: "standardGFamily"},
		api.VMSizeStandardGS5:    {CoreCount: 32, Family: "standardGFamily"},

		api.VMSizeStandardL4s:    {CoreCount: 4, Family: "standardLsv2Family"},
		api.VMSizeStandardL8s:    {CoreCount: 8, Family: "standardLsv2Family"},
		api.VMSizeStandardL16s:   {CoreCount: 16, Family: "standardLsv2Family"},
		api.VMSizeStandardL32s:   {CoreCount: 32, Family: "standardLsv2Family"},
		api.VMSizeStandardL8sV2:  {CoreCount: 8, Family: "standardLsv2Family"},
		api.VMSizeStandardL16sV2: {CoreCount: 16, Family: "standardLsv2Family"},
		api.VMSizeStandardL32sV2: {CoreCount: 32, Family: "standardLsv2Family"},
		api.VMSizeStandardL48sV2: {CoreCount: 48, Family: "standardLsv2Family"},
		api.VMSizeStandardL64sV2: {CoreCount: 64, Family: "standardLsv2Family"},

		// GPU nodes
		// the formatting of the ncasv3_t4 family is different.  This can be seen through a
		// az vm list-usage -l eastus
		api.VMSizeStandardNC4asT4V3:  {CoreCount: 4, Family: "Standard NCASv3_T4 Family"},
		api.VMSizeStandardNC8asT4V3:  {CoreCount: 8, Family: "Standard NCASv3_T4 Family"},
		api.VMSizeStandardNC16asT4V3: {CoreCount: 16, Family: "Standard NCASv3_T4 Family"},
		api.VMSizeStandardNC64asT4V3: {CoreCount: 64, Family: "Standard NCASv3_T4 Family"},
	}

	vm, ok := vmTypesMap[vmSize]
	if !ok {
		return fmt.Errorf("unsupported VMSize %s", vmSize)
	}

	requiredResources["virtualMachines"] += count
	requiredResources["PremiumDiskCount"] += count

	requiredResources[vm.Family] += vm.CoreCount * count
	requiredResources["cores"] += vm.CoreCount * count
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
		if present && int64(required) > (*netUsage.Limit-*netUsage.CurrentValue) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "", "Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.", *netUsage.Name.Value, *netUsage.Limit, *netUsage.CurrentValue, required)
		}
	}

	return nil
}
