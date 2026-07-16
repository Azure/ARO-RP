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
		Name:       new(infraID + apisubnet.NSGSuffixV2),
		Type:       new("Microsoft.Network/networkSecurityGroups"),
		Location:   &location,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		nsg.Properties.SecurityRules = []*armnetwork.SecurityRule{
			{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
					SourcePortRange:          new("*"),
					DestinationPortRange:     new("6443"),
					SourceAddressPrefix:      new("*"),
					DestinationAddressPrefix: new("*"),
					Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
					Priority:                 new(int32(120)),
					Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
				},
				Name: new("apiserver_in"),
			},
		}
	}

	return &arm.Resource{
		Resource:   nsg,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}
