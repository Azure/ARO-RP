package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

type Subnet struct {
	ResourceID string
	IsMaster   bool
}

type Manager interface {
	Get(ctx context.Context, subnetID string) (*mgmtnetwork.Subnet, error)
	GetAll(ctx context.Context, subnetIds []string) ([]*mgmtnetwork.Subnet, error)
	GetHighestFreeIP(ctx context.Context, subnetID string) (string, error)
	CreateOrUpdate(ctx context.Context, subnetID string, subnet *mgmtnetwork.Subnet) error
	CreateOrUpdateFromIds(ctx context.Context, subnetIds []string, gatewayEnabled bool) error
}

type manager struct {
	subnets         network.SubnetsClient
	virtualNetworks network.VirtualNetworksClient
}

func NewManager(environment *azureclient.AROEnvironment, subscriptionID string, spAuthorizer autorest.Authorizer) Manager {
	return &manager{
		subnets:         network.NewSubnetsClient(environment, subscriptionID, spAuthorizer),
		virtualNetworks: network.NewVirtualNetworksClient(environment, subscriptionID, spAuthorizer),
	}
}

// Get retrieves the linked subnet
func (m *manager) Get(ctx context.Context, subnetID string) (*mgmtnetwork.Subnet, error) {
	return m.get(ctx, subnetID, "")
}

func (m *manager) get(ctx context.Context, subnetID, expand string) (*mgmtnetwork.Subnet, error) {
	vnetID, subnetName, err := apisubnet.Split(subnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	subnet, err := m.subnets.Get(ctx, r.ResourceGroup, r.ResourceName, subnetName, expand)
	if err != nil {
		return nil, err
	}

	return &subnet, nil
}

func (m *manager) GetHighestFreeIP(ctx context.Context, subnetID string) (string, error) {
	// Probably anyone who calls this function has a race condition.

	subnet, err := m.get(ctx, subnetID, "ipConfigurations")
	if err != nil {
		return "", err
	}

	_, subnetCIDR, err := net.ParseCIDR(*subnet.AddressPrefix)
	if err != nil {
		return "", err
	}

	bottom, top := cidr.AddressRange(subnetCIDR)

	allocated := map[string]struct{}{}

	// first four addresses and the broadcast address are reserved:
	// https://docs.microsoft.com/en-us/azure/virtual-network/private-ip-addresses#allocation-method
	for i, ip := 0, bottom; i < 4 && !ip.Equal(top); i, ip = i+1, cidr.Inc(ip) {
		allocated[ip.String()] = struct{}{}
	}
	allocated[top.String()] = struct{}{}

	if subnet.IPConfigurations != nil {
		for _, ipconfig := range *subnet.IPConfigurations {
			if ipconfig.PrivateIPAddress != nil {
				allocated[*ipconfig.PrivateIPAddress] = struct{}{}
			}
		}
	}

	for ip := top; !ip.Equal(cidr.Dec(bottom)); ip = cidr.Dec(ip) {
		if _, ok := allocated[ip.String()]; !ok {
			return ip.String(), nil
		}
	}

	return "", nil
}

// CreateOrUpdate updates the linked subnet
func (m *manager) CreateOrUpdate(ctx context.Context, subnetID string, subnet *mgmtnetwork.Subnet) error {
	vnetID, subnetName, err := apisubnet.Split(subnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	return m.subnets.CreateOrUpdateAndWait(ctx, r.ResourceGroup, r.ResourceName, subnetName, *subnet)
}

func (m *manager) GetAll(ctx context.Context, subnetIds []string) ([]*mgmtnetwork.Subnet, error) {
	if len(subnetIds) == 0 {
		return nil, nil
	}

	subnets := make([]*mgmtnetwork.Subnet, len(subnetIds))

	for i, subnetId := range subnetIds {
		subnet, err := m.Get(ctx, subnetId)
		if err != nil {
			return nil, err
		}

		subnets[i] = subnet
	}
	return subnets, nil
}

func (m *manager) CreateOrUpdateFromIds(ctx context.Context, subnetIds []string, gatewayEnabled bool) error {
	subnets, err := m.GetAll(ctx, subnetIds)
	if err != nil {
		return err
	}

	// Only add service endpoints to the subnets if egress lockdown is not enabled.
	if !gatewayEnabled {
		subnetsToBeUpdated := addEndpointsToSubnets(api.SubnetsEndpoints, subnets)

		return m.createOrUpdateSubnets(ctx, subnetsToBeUpdated)
	}

	return m.createOrUpdateSubnets(ctx, subnets)
}

func (m *manager) createOrUpdateSubnets(ctx context.Context, subnets []*mgmtnetwork.Subnet) error {
	for _, subnet := range subnets {
		if err := m.CreateOrUpdate(ctx, *subnet.ID, subnet); err != nil {
			return err
		}
	}
	return nil
}
