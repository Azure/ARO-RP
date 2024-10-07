package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
)

type QuotaValidator interface {
	ValidateQuota(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
}

type quotaValidator struct{}

func addRequiredResources(requiredResources map[string]int, vmSize api.VMSize, count int) error {
	vm, ok := validate.VMSizeFromName(vmSize)
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided VM SKU %s is not supported.", vmSize)
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
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, nil, environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	credential, err := environment.FPNewClientCertificateCredential(tenantID, []string{})
	if err != nil {
		return err
	}
	options := environment.Environment().ArmClientOptions()

	spComputeUsage := compute.NewUsageClient(azEnv, subscriptionID, fpAuthorizer)
	spNetworkUsage, err := armnetwork.NewUsagesClient(subscriptionID, credential, options)
	if err != nil {
		return err
	}

	return validateQuota(ctx, oc, spNetworkUsage, spComputeUsage)
}

func validateQuota(ctx context.Context, oc *api.OpenShiftCluster, spNetworkUsage armnetwork.UsagesClient, spComputeUsage compute.UsageClient) error {
	// If ValidateQuota runs outside install process, we should skip quota validation
	requiredResources := map[string]int{}

	err := addRequiredResources(requiredResources, oc.Properties.MasterProfile.VMSize, 4)
	if err != nil {
		return err
	}

	workerProfiles, _ := api.GetEnrichedWorkerProfiles(oc.Properties)
	//worker node resource calculation
	for _, w := range workerProfiles {
		err := addRequiredResources(requiredResources, w.VMSize, w.Count)
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

	netUsages, err := spNetworkUsage.List(ctx, oc.Location, nil)
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
