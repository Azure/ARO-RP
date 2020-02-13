package privateendpoint

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

const prefix = "rp-pe-"

type Manager interface {
	Create(context.Context, *api.OpenShiftClusterDocument) error
	Delete(context.Context, *api.OpenShiftClusterDocument) error
	GetIP(context.Context, *api.OpenShiftClusterDocument) (string, error)
}

type manager struct {
	env env.Interface

	privateendpoints network.PrivateEndpointsClient
}

func NewManager(env env.Interface, localFPAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		privateendpoints: network.NewPrivateEndpointsClient(env.SubscriptionID(), localFPAuthorizer),
	}
}

func (m *manager) Create(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	return m.privateendpoints.CreateOrUpdateAndWait(ctx, m.env.ResourceGroup(), prefix+doc.ID, mgmtnetwork.PrivateEndpoint{
		PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
			Subnet: &mgmtnetwork.Subnet{
				ID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
			},
			ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr("rp-plsconnection"),
					PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/privateLinkServices/aro-pls"),
					},
				},
			},
		},
		Location: &doc.OpenShiftCluster.Location,
	})
}

func (m *manager) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	return m.privateendpoints.DeleteAndWait(ctx, m.env.ResourceGroup(), prefix+doc.ID)
}

func (m *manager) GetIP(ctx context.Context, doc *api.OpenShiftClusterDocument) (string, error) {
	pe, err := m.privateendpoints.Get(ctx, m.env.ResourceGroup(), prefix+doc.ID, "networkInterfaces")
	if err != nil {
		return "", err
	}

	return *(*(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations)[0].PrivateIPAddress, nil
}
