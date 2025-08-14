package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func (m *manager) clusterNSG(infraID, location string) *arm.Resource {
	nsg := &armnetwork.SecurityGroup{
		Properties: &armnetwork.SecurityGroupPropertiesFormat{},
		Name:       pointerutils.ToPtr(infraID + apisubnet.NSGSuffixV2),
		Type:       pointerutils.ToPtr("Microsoft.Network/networkSecurityGroups"),
		Location:   &location,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		nsg.Properties.SecurityRules = []*armnetwork.SecurityRule{
			{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
					SourcePortRange:          pointerutils.ToPtr("*"),
					DestinationPortRange:     pointerutils.ToPtr("6443"),
					SourceAddressPrefix:      pointerutils.ToPtr("*"),
					DestinationAddressPrefix: pointerutils.ToPtr("*"),
					Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
					Priority:                 pointerutils.ToPtr(int32(120)),
					Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
				},
				Name: pointerutils.ToPtr("apiserver_in"),
			},
		}
	}

	return &arm.Resource{
		Resource:   nsg,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}
