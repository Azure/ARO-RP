package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	armsdk "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) syncPreconfiguredNSG(ctx context.Context) error {
	managedResourceGroup := m.managedResourceGroupName()
	stored := m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG

	subnetIDs, err := m.getSubnetIds()
	if err != nil {
		return err
	}

	inferredValues := map[api.PreconfiguredNSG]string{}
	for _, subnetID := range subnetIDs {
		inferred, nsgID, err := m.inferPreconfiguredNSGForSubnet(ctx, subnetID, managedResourceGroup)
		if err != nil {
			m.log.Warnf("syncPreconfiguredNSG: unable to inspect subnet %s: %v", subnetID, err)
			continue
		}

		if inferred == "" {
			m.log.Warnf("syncPreconfiguredNSG: subnet %s has no attached NSG; cannot infer PreconfiguredNSG (stored value %q)", subnetID, stored)
			continue
		}

		inferredValues[inferred] = nsgID
	}

	switch len(inferredValues) {
	case 0:
		m.log.Warnf("syncPreconfiguredNSG: could not infer PreconfiguredNSG from any subnet; leaving stored value %q unchanged", stored)
		return nil
	case 1:
		// single, unambiguous inferred value
	default:
		m.log.Warnf("syncPreconfiguredNSG: subnets imply conflicting PreconfiguredNSG values %v; leaving stored value %q unchanged", inferredNSGValues(inferredValues), stored)
		return nil
	}

	var inferred api.PreconfiguredNSG
	var nsgID string
	for k, v := range inferredValues {
		inferred, nsgID = k, v
	}

	if inferred == stored {
		m.log.Infof("syncPreconfiguredNSG: PreconfiguredNSG is consistent (%q)", stored)
		return nil
	}

	m.log.Warnf("syncPreconfiguredNSG: PreconfiguredNSG mismatch: stored %q but attached NSG %q implies %q; updating stored value", stored, nsgID, inferred)

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG = inferred
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure updating PreconfiguredNSG to %q: %w", inferred, err)
	}

	m.log.Infof("syncPreconfiguredNSG: PreconfiguredNSG updated to %q", inferred)
	return nil
}

func (m *manager) inferPreconfiguredNSGForSubnet(ctx context.Context, subnetID, managedResourceGroup string) (api.PreconfiguredNSG, string, error) {
	r, err := armsdk.ParseResourceID(subnetID)
	if err != nil {
		return "", "", err
	}

	subnetResponse, err := m.armSubnets.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
	if err != nil {
		return "", "", err
	}
	s := subnetResponse.Subnet

	if s.Properties == nil || s.Properties.NetworkSecurityGroup == nil || s.Properties.NetworkSecurityGroup.ID == nil {
		return "", "", nil
	}

	nsgID := *s.Properties.NetworkSecurityGroup.ID
	nsg, err := armsdk.ParseResourceID(nsgID)
	if err != nil {
		return "", nsgID, fmt.Errorf("unable to parse NSG ID %q: %w", nsgID, err)
	}

	if strings.EqualFold(nsg.ResourceGroupName, managedResourceGroup) {
		return api.PreconfiguredNSGDisabled, nsgID, nil
	}
	return api.PreconfiguredNSGEnabled, nsgID, nil
}

func inferredNSGValues(m map[api.PreconfiguredNSG]string) []api.PreconfiguredNSG {
	values := make([]api.PreconfiguredNSG, 0, len(m))
	for k := range m {
		values = append(values, k)
	}
	return values
}
