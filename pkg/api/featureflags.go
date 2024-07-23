package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// FeatureFlagMTU3900 is the feature in the subscription that causes new
	// OpenShift cluster nodes to use the largest available Maximum Transmission
	// Unit (MTU) on Azure virtual networks, which as of late 2021 is 3900 bytes.
	// Otherwise cluster nodes will use the DHCP-provided MTU of 1500 bytes.
	FeatureFlagMTU3900 = "Microsoft.RedHatOpenShift/MTU3900"
)
