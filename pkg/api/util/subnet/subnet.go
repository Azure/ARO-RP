package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
)

type Subnet struct {
	ResourceID string
	IsMaster   bool
}

// Split splits the given subnetID into a vnetID and subnetName
func Split(subnetID string) (string, string, error) {
	parts := strings.Split(subnetID, "/")
	if len(parts) != 11 {
		return "", "", fmt.Errorf("subnet ID %q has incorrect length", subnetID)
	}

	return strings.Join(parts[:len(parts)-2], "/"), parts[len(parts)-1], nil
}

// NetworkSecurityGroupID returns the NetworkSecurityGroup ID for a given subnet ID
func NetworkSecurityGroupID(oc *api.OpenShiftCluster, subnetID string) (string, error) {
	infraID := oc.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}
	isWorkerSubnet := false
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(oc.Properties)

	for _, s := range workerProfiles {
		if strings.EqualFold(subnetID, s.SubnetID) {
			isWorkerSubnet = true
			break
		}
	}
	return NetworkSecurityGroupIDExpanded(oc.Properties.ArchitectureVersion, oc.Properties.ClusterProfile.ResourceGroupID, infraID, isWorkerSubnet)
}

// NetworkSecurityGroupIDExpanded returns the NetworkSecurityGroup ID for a given subnetID, without the OpenShift Cluster document
func NetworkSecurityGroupIDExpanded(architectureVersion api.ArchitectureVersion, resourceGroupID, infraID string, isWorkerSubnet bool) (string, error) {
	switch architectureVersion {
	case api.ArchitectureVersionV1:
		return networkSecurityGroupIDV1(resourceGroupID, infraID, isWorkerSubnet), nil
	case api.ArchitectureVersionV2:
		return networkSecurityGroupIDV2(resourceGroupID, infraID), nil
	default:
		return "", fmt.Errorf("unknown architecture version %d", architectureVersion)
	}
}

func networkSecurityGroupIDV1(resourceGroupID, infraID string, isWorkerSubnet bool) string {
	if isWorkerSubnet {
		return resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/" + infraID + NSGNodeSuffixV1
	}
	return resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/" + infraID + NSGControlPlaneSuffixV1
}

func networkSecurityGroupIDV2(resourceGroupID, infraID string) string {
	return resourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/" + infraID + NSGSuffixV2
}
