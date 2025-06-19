package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (m *manager) clusterNSG(infraID, location string) *arm.Resource {
	nsg := &mgmtnetwork.SecurityGroup{
		SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		Name:                          to.Ptr(infraID + apisubnet.NSGSuffixV2),
		Type:                          to.Ptr("Microsoft.Network/networkSecurityGroups"),
		Location:                      &location,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		nsg.SecurityRules = &[]mgmtnetwork.SecurityRule{
			{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.Ptr("*"),
					DestinationPortRange:     to.Ptr("6443"),
					SourceAddressPrefix:      to.Ptr("*"),
					DestinationAddressPrefix: to.Ptr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 to.Int32Ptr(120),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.Ptr("apiserver_in"),
			},
		}
	}

	return &arm.Resource{
		Resource:   nsg,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}
