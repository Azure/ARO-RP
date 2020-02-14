package virtualnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

const prefix = "rp-pe-"

type Manager interface {
	Create(ctx context.Context, virtualNetworkName string, addressSpace string, networkSecurityGroupID string) error
	//Delete(ctx context.Context, name string) error
	//GetFreeIPs(ctx context.Context, name string) (string, error)
}

type manager struct {
	env env.Interface

	virtualnetworks network.VirtualNetworksClient
}

func NewManager(env env.Interface, localFPAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		virtualnetworks: network.NewVirtualNetworksClient(env.SubscriptionID(), localFPAuthorizer),
	}
}

func (m *manager) Create(ctx context.Context, virtualNetworkName string, addressSpace string, networkSecurityGroupID string) error {
	return m.virtualnetworks.CreateOrUpdateAndWait(ctx, m.env.ResourceGroup(), prefix+virtualNetworkName, mgmtnetwork.VirtualNetwork{
		Name: to.StringPtr(prefix + virtualNetworkName),
		VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &mgmtnetwork.AddressSpace{
				AddressPrefixes: &[]string{addressSpace},
			},
			Subnets: &[]mgmtnetwork.Subnet{
				{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(addressSpace),
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr(networkSecurityGroupID),
						},
					},
				},
			},
		},
		Location: to.StringPtr(m.env.Location()),
	})
}

func (m *manager) Delete(ctx context.Context, virtualNetworkName string) error {
	return m.virtualnetworks.DeleteAndWait(ctx, m.env.ResourceGroup(), prefix+virtualNetworkName)
}
