package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// FeatureFlagSaveAROTestConfig is the feature in the subscription that is used
	// to indicate if we need to save ARO cluster config into the E2E
	// StorageAccount
	FeatureFlagSaveAROTestConfig = "Microsoft.RedHatOpenShift/SaveAROTestConfig"

	// FeatureFlagAdminKubeconfig is the feature in the subscription that is used
	// to enable adminKubeconfig api. API itself returns privileged kubeconfig.
	// We need a feature flag to make sure we don't open a security hole in existing
	// clusters before customer had a chance to patch their API RBAC
	FeatureFlagAdminKubeconfig = "Microsoft.RedHatOpenShift/AdminKubeconfig"

	// FeatureFlagMTU3900 is the feature in the subscription that causes new
	// OpenShift cluster nodes to use the largest available Maximum Transmission
	// Unit (MTU) on Azure virtual networks, which as of late 2021 is 3900 bytes.
	// Otherwise cluster nodes will use the DHCP-provided MTU of 1500 bytes.
	FeatureFlagMTU3900 = "Microsoft.RedHatOpenShift/MTU3900"
)
