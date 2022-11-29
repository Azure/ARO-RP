package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

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

type QuotaValidator interface {
	ValidateQuota(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
	//ValidateNewCluster(ctx context.Context, envValue env.Interface, subscription *api.SubscriptionDocument, cluster *api.OpenShiftCluster, staticValidator api.OpenShiftClusterStaticValidator, ext interface{}, path string) error
}

type quotaValidator struct{}

func addRequiredResources(requiredResources map[string]int, vmSize api.VMSize, count int) error {
	vmTypesMap := map[api.VMSize]struct {
		CoreCount int
		Family    string
	}{
		api.VMSizeStandardD2sV3: {CoreCount: 2, Family: standardDSv3},

		api.VMSizeStandardD4asV4:  {CoreCount: 4, Family: standardDASv4},
		api.VMSizeStandardD8asV4:  {CoreCount: 8, Family: standardDASv4},
		api.VMSizeStandardD16asV4: {CoreCount: 16, Family: standardDASv4},
		api.VMSizeStandardD32asV4: {CoreCount: 32, Family: standardDASv4},

		api.VMSizeStandardD4sV3:  {CoreCount: 4, Family: standardDSv3},
		api.VMSizeStandardD8sV3:  {CoreCount: 8, Family: standardDSv3},
		api.VMSizeStandardD16sV3: {CoreCount: 16, Family: standardDSv3},
		api.VMSizeStandardD32sV3: {CoreCount: 32, Family: standardDSv3},

		api.VMSizeStandardE4sV3:     {CoreCount: 4, Family: standardESv3},
		api.VMSizeStandardE8sV3:     {CoreCount: 8, Family: standardESv3},
		api.VMSizeStandardE16sV3:    {CoreCount: 16, Family: standardESv3},
		api.VMSizeStandardE32sV3:    {CoreCount: 32, Family: standardESv3},
		api.VMSizeStandardE64isV3:   {CoreCount: 64, Family: standardESv3},
		api.VMSizeStandardE64iV3:    {CoreCount: 64, Family: standardESv3},
		api.VMSizeStandardE80isV4:   {CoreCount: 80, Family: standardEISv4},
		api.VMSizeStandardE80idsV4:  {CoreCount: 80, Family: standardEIDSv4},
		api.VMSizeStandardE104iV5:   {CoreCount: 104, Family: standardEIv5},
		api.VMSizeStandardE104isV5:  {CoreCount: 104, Family: standardEISv5},
		api.VMSizeStandardE104idV5:  {CoreCount: 104, Family: standardEIDv5},
		api.VMSizeStandardE104idsV5: {CoreCount: 104, Family: standardEIDSv5},

		api.VMSizeStandardF4sV2:  {CoreCount: 4, Family: standardFSv2},
		api.VMSizeStandardF8sV2:  {CoreCount: 8, Family: standardFSv2},
		api.VMSizeStandardF16sV2: {CoreCount: 16, Family: standardFSv2},
		api.VMSizeStandardF32sV2: {CoreCount: 32, Family: standardFSv2},
		api.VMSizeStandardF72sV2: {CoreCount: 72, Family: standardFSv2},

		api.VMSizeStandardM128ms: {CoreCount: 128, Family: standardMS},
		api.VMSizeStandardG5:     {CoreCount: 32, Family: standardGFamily},
		api.VMSizeStandardGS5:    {CoreCount: 32, Family: standardGFamily},

		api.VMSizeStandardL4s:    {CoreCount: 4, Family: standardLSv2},
		api.VMSizeStandardL8s:    {CoreCount: 8, Family: standardLSv2},
		api.VMSizeStandardL16s:   {CoreCount: 16, Family: standardLSv2},
		api.VMSizeStandardL32s:   {CoreCount: 32, Family: standardLSv2},
		api.VMSizeStandardL8sV2:  {CoreCount: 8, Family: standardLSv2},
		api.VMSizeStandardL16sV2: {CoreCount: 16, Family: standardLSv2},
		api.VMSizeStandardL32sV2: {CoreCount: 32, Family: standardLSv2},
		api.VMSizeStandardL48sV2: {CoreCount: 48, Family: standardLSv2},
		api.VMSizeStandardL64sV2: {CoreCount: 64, Family: standardLSv2},

		// GPU nodes
		// the formatting of the ncasv3_t4 family is different.  This can be seen through a
		// az vm list-usage -l eastus
		api.VMSizeStandardNC4asT4V3:  {CoreCount: 4, Family: standardNCAS},
		api.VMSizeStandardNC8asT4V3:  {CoreCount: 8, Family: standardNCAS},
		api.VMSizeStandardNC16asT4V3: {CoreCount: 16, Family: standardNCAS},
		api.VMSizeStandardNC64asT4V3: {CoreCount: 64, Family: standardNCAS},
	}

	vm, ok := vmTypesMap[vmSize]
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorUnsupportedSKU, "", "The provided VM SKU %s is not supported.", vmSize)
	}

	requiredResources["virtualMachines"] += count
	requiredResources["PremiumDiskCount"] += count

	requiredResources[vm.Family] += vm.CoreCount * count
	requiredResources["cores"] += vm.CoreCount * count
	return nil
}

// ValidateQuota checks usage quotas vs. resources required by cluster before cluster
// creation
// It is a method on struct so we can make use of interfaces.
func (q quotaValidator) ValidateQuota(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error {
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, environment.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spComputeUsage := compute.NewUsageClient(azEnv, subscriptionID, fpAuthorizer)
	spNetworkUsage := network.NewUsageClient(azEnv, subscriptionID, fpAuthorizer)

	return validateQuota(ctx, oc, spNetworkUsage, spComputeUsage)
}

func validateQuota(ctx context.Context, oc *api.OpenShiftCluster, spNetworkUsage network.UsageClient, spComputeUsage compute.UsageClient) error {
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
	computeUsages, err := spComputeUsage.List(ctx, oc.Location)
	if err != nil {
		return err
	}

	for _, usage := range computeUsages {
		required, present := requiredResources[*usage.Name.Value]
		if present && int64(required) > (*usage.Limit-int64(*usage.CurrentValue)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "", "Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.", *usage.Name.Value, *usage.Limit, *usage.CurrentValue, required)
		}
	}

	netUsages, err := spNetworkUsage.List(ctx, oc.Location)
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
