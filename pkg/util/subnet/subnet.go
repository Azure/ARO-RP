package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

type Manager interface {
	Get(ctx context.Context, subnetID string) (*mgmtnetwork.Subnet, error)
	CreateOrUpdate(ctx context.Context, subnetID string, subnet *mgmtnetwork.Subnet) error
}

type manager struct {
	subnets network.SubnetsClient
}

func NewManager(subscriptionID string, spAuthorizer autorest.Authorizer) Manager {
	return &manager{
		subnets: network.NewSubnetsClient(subscriptionID, spAuthorizer),
	}
}

// Get retrieves the linked subnet
func (m *manager) Get(ctx context.Context, subnetID string) (*mgmtnetwork.Subnet, error) {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	subnet, err := m.subnets.Get(ctx, r.ResourceGroup, r.ResourceName, subnetName, "")
	if err != nil {
		return nil, err
	}

	return &subnet, nil
}

// CreateOrUpdate updates the linked subnet
func (m *manager) CreateOrUpdate(ctx context.Context, subnetID string, subnet *mgmtnetwork.Subnet) error {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	return m.subnets.CreateOrUpdateAndWait(ctx, r.ResourceGroup, r.ResourceName, subnetName, *subnet)
}

// Split splits the given subnetID into a vnetID and subnetName
func Split(subnetID string) (string, string, error) {
	parts := strings.Split(subnetID, "/")
	if len(parts) != 11 {
		return "", "", fmt.Errorf("subnet ID %q has incorrect length", subnetID)
	}

	return strings.Join(parts[:len(parts)-2], "/"), parts[len(parts)-1], nil
}

// NetworkSecurityGroupID returns the NetworkSecurityGroup ID for a given subnet
// ID
func NetworkSecurityGroupID(oc *api.OpenShiftCluster, subnetID string) (string, error) {
	switch {
	case strings.EqualFold(subnetID, oc.Properties.MasterProfile.SubnetID):
		return oc.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg", nil
	case strings.EqualFold(subnetID, oc.Properties.WorkerProfiles[0].SubnetID):
		return oc.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg", nil
	default:
		return "", fmt.Errorf("unknown subnetID %q", subnetID)
	}
}
