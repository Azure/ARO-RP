package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *manager) preUpgradeChecks(ctx context.Context) error {
	return checkCustomDNS(ctx, i.doc, i.vnet)
}

// checkCustomDNS checks if customer has custom DNS configured on VNET.
// This would cause nodes to rotate and render cluster inoperable
func checkCustomDNS(ctx context.Context, doc *api.OpenShiftClusterDocument, vnet network.VirtualNetworksClient) error {
	infraID := doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	vnetID, _, err := subnet.Split(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	v, err := vnet.Get(ctx, r.ResourceGroup, r.ResourceName, "")
	if err != nil {
		return err
	}

	if v.VirtualNetworkPropertiesFormat.DhcpOptions != nil &&
		v.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers != nil &&
		len(*v.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers) > 0 {
		return fmt.Errorf("not upgrading: custom DNS is set")
	}

	return nil
}
