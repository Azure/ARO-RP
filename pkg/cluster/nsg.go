package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func (m *manager) clusterNSG(infraID, location string) *arm.Resource {
	nsg := &mgmtnetwork.SecurityGroup{
		SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		Name:                          pointerutils.ToPtr(infraID + apisubnet.NSGSuffixV2),
		Type:                          pointerutils.ToPtr("Microsoft.Network/networkSecurityGroups"),
		Location:                      &location,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		nsg.SecurityRules = &[]mgmtnetwork.SecurityRule{
			{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          pointerutils.ToPtr("*"),
					DestinationPortRange:     pointerutils.ToPtr("6443"),
					SourceAddressPrefix:      pointerutils.ToPtr("*"),
					DestinationAddressPrefix: pointerutils.ToPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 pointerutils.ToPtr(int32(120)),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
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
