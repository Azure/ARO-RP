package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (m *manager) removePrivateDNSZone(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	zones, err := m.privateZones.ListByResourceGroup(ctx, resourceGroup, nil)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	if len(zones) == 0 {
		// fix up any clusters that we already upgraded
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			dns, err := m.configcli.ConfigV1().DNSes().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}

			if dns.Spec.PrivateZone == nil ||
				!strings.HasPrefix(strings.ToLower(dns.Spec.PrivateZone.ID), strings.ToLower(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)) {
				return nil
			}

			dns.Spec.PrivateZone = nil

			_, err = m.configcli.ConfigV1().DNSes().Update(ctx, dns, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			m.log.Print(err)
		}

		return nil
	}

	mcps, err := m.mcocli.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
		return nil
	}

	var machineCount int
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

		machineCount += int(mcp.Status.MachineCount)
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
		return nil
	}

	if len(nodes.Items) != machineCount {
		m.log.Printf("cluster has %d nodes but %d under MCPs, not removing private DNS zone", len(nodes.Items), machineCount)
		return nil
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return err
	}
	v, err := version.GetClusterVersion(cv)
	if err != nil {
		m.log.Print(err)
		return nil
	}

	if v.Lt(version.NewVersion(4, 4)) {
		// 4.3 uses SRV records for etcd
		m.log.Printf("cluster version < 4.4, not removing private DNS zone")
		return nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dns, err := m.configcli.ConfigV1().DNSes().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if dns.Spec.PrivateZone == nil ||
			!strings.HasPrefix(strings.ToLower(dns.Spec.PrivateZone.ID), strings.ToLower(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)) {
			return nil
		}

		dns.Spec.PrivateZone = nil

		_, err = m.configcli.ConfigV1().DNSes().Update(ctx, dns, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		m.log.Print(err)
		return nil
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
