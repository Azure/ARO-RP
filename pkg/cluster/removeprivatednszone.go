package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) removePrivateDNSZone(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	zones, err := m.privateZones.ListByResourceGroup(ctx, resourceGroup, nil)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	if len(zones) == 0 {
		return nil
	}

	mcps, err := m.mcocli.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
		return nil
	}

	for _, mcp := range mcps.Items {
		var found bool
		for _, source := range mcp.Status.Configuration.Source {
			if source.Name == "99-"+mcp.Name+"-aro-dns" {
				found = true
				break
			}
		}

		if !found {
			m.log.Printf("ARO DNS config not found in MCP %s", mcp.Name)
			return nil
		}

		if !ready.MachineConfigPoolIsReady(&mcp) {
			m.log.Printf("MCP %s not ready", mcp.Name)
			return nil
		}
	}

	for _, zone := range zones {
		err = m.deletePrivateDNSVirtualNetworkLinks(ctx, *zone.ID)
		if err != nil {
			m.log.Print(err)
			return nil
		}

		r, err := azure.ParseResourceID(*zone.ID)
		if err != nil {
			m.log.Print(err)
			return nil
		}

		err = m.privateZones.DeleteAndWait(ctx, resourceGroup, r.ResourceName, "")
		if err != nil {
			m.log.Print(err)
			return nil
		}
	}

	return nil
}
