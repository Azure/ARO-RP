package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) fixNSG(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		return nil
	}

	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	nsg, err := m.securitygroups.Get(ctx, resourceGroup, infraID+subnet.NSGControlPlaneSuffix, "")
	if err != nil {
		return err
	}

	if nsg.SecurityGroupPropertiesFormat == nil ||
		nsg.SecurityRules == nil {
		return nil
	}

	rules := make([]mgmtnetwork.SecurityRule, 0, len(*nsg.SecurityRules))

	for _, rule := range *nsg.SecurityGroupPropertiesFormat.SecurityRules {
		if rule.SecurityRulePropertiesFormat != nil &&
			rule.Protocol == mgmtnetwork.SecurityRuleProtocolTCP &&
			rule.DestinationPortRange != nil &&
			*rule.DestinationPortRange == "6443" {
			continue
		}

		rules = append(rules, rule)
	}

	if len(rules) == len(*nsg.SecurityRules) {
		return nil
	}

	nsg.SecurityRules = &rules

	return m.securitygroups.CreateOrUpdateAndWait(ctx, resourceGroup, infraID+subnet.NSGControlPlaneSuffix, nsg)
}
