package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// FeatureFlagSaveAROTestConfig is the feature in the subscription that is used
	// to indicate if we need to save ARO cluster config into the E2E
	// StorageAccount
	FeatureFlagSaveAROTestConfig = "Microsoft.RedHatOpenShift/SaveAROTestConfig"

	// FeatureFlagMTU3900 is the feature in the subscription that causes new
	// OpenShift cluster nodes to use the largest available Maximum Transmission
	// Unit (MTU) on Azure virtual networks, which as of late 2021 is 3900 bytes.
	// Otherwise cluster nodes will use the DHCP-provided MTU of 1500 bytes.
	FeatureFlagMTU3900 = "Microsoft.RedHatOpenShift/MTU3900"

	// FeatureFlagUserDefinedRouting is the feature in the subscription that is used to indicate we need to
	// provision a private cluster without an IP address
	FeatureFlagUserDefinedRouting = "Microsoft.RedHatOpenShift/UserDefinedRouting"

	// FeatureFlagCheckAccessTestToggle is used for safely testing the new check access
	// API in production. The toggle will be removed once the testing has been completed.
	FeatureFlagCheckAccessTestToggle = "Microsoft.RedHatOpenShift/CheckAccessTestToggle"

	// FeatureFlagBYONsg is used for indicating whether a customer subscription
	// is registered for BYO NSG feature.
	FeatureFlagBYONsg = "Microsoft.RedHatOpenShift/BYONsg"
)
