package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (m *manager) clusterNSG(infraID, location string) *arm.Resource {
	nsg := &mgmtnetwork.SecurityGroup{
		SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		Name:                          to.StringPtr(infraID + apisubnet.NSGSuffixV2),
		Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location:                      &location,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		nsg.SecurityRules = &[]mgmtnetwork.SecurityRule{
			{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("6443"),
					SourceAddressPrefix:      to.StringPtr("*"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 to.Int32Ptr(120),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("apiserver_in"),
			},
		}
	}

	return &arm.Resource{
		Resource:   nsg,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}
