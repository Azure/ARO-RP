package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

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
	CreateOrUpdate(ctx context.Context, subnetID string, subnet *mgmtnetwork.Subnet) error
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
