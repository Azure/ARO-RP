package privateendpoint

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const rpPEPrefix = "rp-pe-"

type Manager interface {
	CreateRP(context.Context, *api.OpenShiftClusterDocument) error
	DeleteRP(context.Context, *api.OpenShiftClusterDocument) error
	// GetIPRP return single IP address, used in RP-> Cluster communication
	GetIPRP(context.Context, *api.OpenShiftClusterDocument) (string, error)

	// GetIPsACR return map of IP addresses, used in Cluster-> ACR communication
	GetIPsACR(context.Context, *api.OpenShiftClusterDocument) (map[string]string, error)
}

type manager struct {
	subscriptionID string
	resourceGroup  string

	privateendpoints network.PrivateEndpointsClient
}

func NewManager(subscriptionID, resourceGroup string, authorizer autorest.Authorizer) Manager {
	return &manager{
		subscriptionID:   subscriptionID,
		resourceGroup:    resourceGroup,
		privateendpoints: network.NewPrivateEndpointsClient(subscriptionID, authorizer),
	}
}

func (m *manager) CreateRP(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	infraID := doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	return m.create(ctx, rpPEPrefix+doc.ID, m.resourceGroup, mgmtnetwork.PrivateEndpoint{
		PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
			Subnet: &mgmtnetwork.Subnet{
				ID: to.StringPtr("/subscriptions/" + m.subscriptionID + "/resourceGroups/" + m.resourceGroup + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
			},
			ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr("rp-plsconnection"),
					PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/privateLinkServices/" + infraID + "-pls"),
					},
				},
			},
		},
		Location: &doc.OpenShiftCluster.Location,
	})
}

func (m *manager) DeleteRP(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	return m.delete(ctx, rpPEPrefix+doc.ID, m.resourceGroup)
}

func (m *manager) GetIPRP(ctx context.Context, doc *api.OpenShiftClusterDocument) (string, error) {
	ips, err := m.getIPs(ctx, rpPEPrefix+doc.ID, m.resourceGroup)
	if err != nil {
		return "", err
	}
	return *ips[0].PrivateIPAddress, nil
}

func (m *manager) GetIPsACR(ctx context.Context, doc *api.OpenShiftClusterDocument) (map[string]string, error) {
	resourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	ips, err := m.getIPs(ctx, infraID+"-arosvc-pe", resourceGroup)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(ips))
	for _, ip := range ips {
		if ip.InterfaceIPConfigurationPropertiesFormat != nil &&
			ip.InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress != nil &&
			ip.Name != nil {
			result[*ip.Name] = *ip.InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress
		}
	}

	return result, nil
}

func (m *manager) create(ctx context.Context, name, resourceGroup string, pe mgmtnetwork.PrivateEndpoint) error {
	return m.privateendpoints.CreateOrUpdateAndWait(ctx, resourceGroup, name, pe)
}

func (m *manager) delete(ctx context.Context, name, resourceGroup string) error {
	return m.privateendpoints.DeleteAndWait(ctx, resourceGroup, name)
}

func (m *manager) getIPs(ctx context.Context, name, resourceGroup string) ([]mgmtnetwork.InterfaceIPConfiguration, error) {
	pe, err := m.privateendpoints.Get(ctx, resourceGroup, name, "networkInterfaces")
	if err != nil {
		return nil, err
	}

	return *(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations, nil
}
