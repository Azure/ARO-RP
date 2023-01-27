package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/azure"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	v1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
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
		if err := utilnet.UpdateDNSs(ctx, m.configcli.ConfigV1().DNSes(), m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID); err != nil {
			m.log.Print(err)
		}
		return nil
	}

	if !m.clusterHasSameNumberOfNodesAndMachineConfigPools(ctx) {
		return nil
	}

	if !version.ClusterVersionIsGreaterThan4_3(ctx, m.configcli, m.log) {
		return nil
	}

	if err = utilnet.UpdateDNSs(ctx, m.configcli.ConfigV1().DNSes(), m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID); err != nil {
		m.log.Print(err)
		return nil
	}

	utilnet.RemoveZones(ctx, m.log, m.virtualNetworkLinks, m.privateZones, zones, resourceGroup)
	return nil
}

func (m *manager) clusterHasSameNumberOfNodesAndMachineConfigPools(ctx context.Context) bool {
	machineConfigPoolList, err := m.mcocli.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
		return false
	}

	nMachineConfigPools, errorOcurred := validateMachineConfigPoolsAndGetCounter(machineConfigPoolList.Items, m.log)
	if errorOcurred {
		return false
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
		return false
	}

	nNodes := len(nodes.Items)
	if nNodes != nMachineConfigPools {
		m.log.Printf("cluster has %d nodes but %d under MCPs, not removing private DNS zone", nNodes, nMachineConfigPools)
		return false
	}
	return true
}

func validateMachineConfigPoolsAndGetCounter(machineConfigPools []mcv1.MachineConfigPool, logEntry *logrus.Entry) (nMachineConfigPools int, errOccurred bool) {
	for _, mcp := range machineConfigPools {
		if !utilnet.McpContainsARODNSConfig(mcp) {
			logEntry.Printf("ARO DNS config not found in MCP %s", mcp.Name)
			return 0, true
		}

		if !ready.MachineConfigPoolIsReady(&mcp) {
			logEntry.Printf("MCP %s not ready", mcp.Name)
			return 0, true
		}

		nMachineConfigPools += int(mcp.Status.MachineCount)
	}
	return nMachineConfigPools, false
}

func mcpContainsARODNSConfig(mcp mcv1.MachineConfigPool) bool {
	for _, source := range mcp.Status.Configuration.Source {
		mcpPrefix := "99-"
		mcpSuffix := "-aro-dns"

		if source.Name == mcpPrefix+mcp.Name+mcpSuffix {
			return true
		}
	}
	return false
}

func (m *manager) updateDNSs(ctx context.Context) error {
	fn := updateClusterDNSFn(ctx, m.configcli.ConfigV1().DNSes(), m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	return retry.RetryOnConflict(retry.DefaultRetry, fn)
}

func updateClusterDNSFn(ctx context.Context, dnsInterface v1.DNSInterface, resourceGroupID string) func() error {
	return func() error {
		dns, err := dnsInterface.Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if dns.Spec.PrivateZone == nil ||
			!strings.HasPrefix(
				strings.ToLower(dns.Spec.PrivateZone.ID),
				strings.ToLower(resourceGroupID)) {
			return nil
		}

		dns.Spec.PrivateZone = nil

		_, err = dnsInterface.Update(ctx, dns, metav1.UpdateOptions{})
		return err
	}
}

func clusterVersionIsAtLeast4_4(ctx context.Context, configcli configclient.Interface, logEntry *logrus.Entry) (bool, error) {
	cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	v, err := version.GetClusterVersion(cv)
	if err != nil {
		logEntry.Print(err)
		return false, nil
	}

	if v.Lt(version.NewVersion(4, 4)) {
		// 4.3 uses SRV records for etcd
		logEntry.Printf("cluster version < 4.4, not removing private DNS zone")
		return false, nil
	}
	return true, nil
}

func (m *manager) removeZones(ctx context.Context, privateZones []mgmtprivatedns.PrivateZone, resourceGroup string) {
	for _, privateZone := range privateZones {
		if err := utilnet.DeletePrivateDNSVirtualNetworkLinks(ctx, m.virtualNetworkLinks, *privateZone.ID); err != nil {
			m.log.Print(err)
			return
		}

		r, err := azure.ParseResourceID(*privateZone.ID)
		if err != nil {
			m.log.Print(err)
			return
		}

		if err = m.privateZones.DeleteAndWait(ctx, resourceGroup, r.ResourceName, ""); err != nil {
			m.log.Print(err)
			return
		}
	}
}
