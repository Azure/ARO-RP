package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const outboundRuleV4 = "outbound-rule-v4"

func (m *manager) reconcileLoadBalancerProfile(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType != api.OutboundTypeLoadbalancer {
		return nil
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	lb, err := m.loadBalancers.Get(ctx, resourceGroupName, infraID, "")
	if err != nil {
		return err
	}

	err = m.reconcileOutboundRuleV4IPs(ctx, lb)
	if err != nil {
		return err
	}

	return nil
}

// Reconcile the outbound rule "outbound-rule-v4" frontend IP Config.
func (m *manager) reconcileOutboundRuleV4IPs(ctx context.Context, lb mgmtnetwork.LoadBalancer) error {
	m.log.Info("reconciling outbound-rule-v4")
	defer func() {
		err := m.deleteUnusedManagedIPs(ctx)
		if err != nil {
			m.log.Error("failed to cleanup unused managed IPs, error: %w", err)
		}
	}()

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	originalOutboundIPs := getOutboundIPsFromLB(lb)
	// ensure effectiveOutboundIPs is patched the first time running against a cluster
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs == nil {
		err := m.patchEffectiveOutboundIPs(ctx, originalOutboundIPs)
		if err != nil {
			return err
		}
	}

	desiredOutboundIPs, err := m.getDesiredOutboundIPs(ctx)
	if err != nil {
		return err
	}

	if areResourceRefsEqual(desiredOutboundIPs, originalOutboundIPs) {
		return nil
	}

	// rebuild outbound-rule-v4 frontend ip config with desired outbound ips
	removeOutboundIPsFromLB(lb)
	addOutboundIPsToLB(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, lb, desiredOutboundIPs)

	err = m.loadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID, lb)
	if err != nil {
		return err
	}

	// update database with new effective outbound IPs
	err = m.patchEffectiveOutboundIPs(ctx, desiredOutboundIPs)
	if err != nil {
		return err
	}

	return nil
}

// Remove all frontend ip config in use by outbound-rule-v4.  Frontend IP config that is used by load balancer rules will be saved.
func removeOutboundIPsFromLB(lb mgmtnetwork.LoadBalancer) {
	// get all outbound rule fip config to remove
	var obRuleV4FIPConfigs = make(map[string]mgmtnetwork.SubResource)
	for _, obRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *obRule.Name == outboundRuleV4 {
			for i := 0; i < len(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations); i++ {
				fipConfigID := *(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i].ID
				fipConfig := (*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i]
				obRuleV4FIPConfigs[fipConfigID] = fipConfig
			}
			// clear outbound-rule-v4 frontend ip config
			*obRule.FrontendIPConfigurations = []mgmtnetwork.SubResource{}
			break
		}
	}

	// rebuild frontend ip config without outbound-rule-v4 frontend ip config, preserving
	// the public api server frontend ip config if the api server is public
	var savedFIPConfig = make([]mgmtnetwork.FrontendIPConfiguration, 0, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))
	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		fipLBRules := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].LoadBalancingRules
		if _, ok := obRuleV4FIPConfigs[fipConfigID]; ok && fipLBRules == nil {
			continue
		}
		savedFIPConfig = append(savedFIPConfig, fipConfig)
	}
	lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &savedFIPConfig
}

// return a map of Frontend IP Configs where the key is the ID of the Frontend IP Config
func getFrontendIPConfigs(lb mgmtnetwork.LoadBalancer) map[string]mgmtnetwork.FrontendIPConfiguration {
	// map out frontendConfig to ID of public IP addresses for quick lookup
	var frontendIPConfigsMap = make(map[string]mgmtnetwork.FrontendIPConfiguration, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigIPID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].FrontendIPConfigurationPropertiesFormat.PublicIPAddress.ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		frontendIPConfigsMap[fipConfigIPID] = fipConfig
	}

	return frontendIPConfigsMap
}

// Adds IPs or IPPrefixes to the load balancer outbound rule "outbound-rule-v4".
func addOutboundIPsToLB(resourceGroupID string, lb mgmtnetwork.LoadBalancer, obIPsOrIPPrefixes []api.ResourceReference) {
	frontendIPConfigs := getFrontendIPConfigs(lb)
	outboundRuleV4FrontendIPConfig := []mgmtnetwork.SubResource{}

	// add IP Addresses to frontendConfig
	for _, obIPOrPrefix := range obIPsOrIPPrefixes {
		// check if the frontend config exists in the map to avoid duplicate entries
		if _, ok := frontendIPConfigs[obIPOrPrefix.ID]; !ok {
			frontendIPConfigName := stringutils.LastTokenByte(obIPOrPrefix.ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations, newFrontendIPConfig(frontendIPConfigName, frontendConfigID, obIPOrPrefix.ID))
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(frontendConfigID))
		} else {
			// frontendIPConfig already exists and just needs to be added to the outbound rule
			frontendConfig := frontendIPConfigs[obIPOrPrefix.ID]
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(*frontendConfig.ID))
		}
	}

	// update outbound-rule-v4
	for _, outboundRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *outboundRule.Name == outboundRuleV4 {
			outboundRule.OutboundRulePropertiesFormat.FrontendIPConfigurations = &outboundRuleV4FrontendIPConfig
			break
		}
	}
}

// Delete all managed outbound IPs that are not in use by the load balancer.
// The default outbound ip is saved if the api server is public.
func (m *manager) deleteUnusedManagedIPs(ctx context.Context) error {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	managedIPs, err := m.getClusterManagedIPs(ctx)
	if err != nil {
		return err
	}

	lb, err := m.loadBalancers.Get(ctx, resourceGroupName, infraID, "")
	if err != nil {
		return err
	}

	outboundIPs := getOutboundIPsFromLB(lb)
	outboundIPMap := make(map[string]api.ResourceReference, len(outboundIPs))
	for i := 0; i < len(outboundIPs); i++ {
		outboundIPMap[strings.ToLower(outboundIPs[i].ID)] = outboundIPs[i]
	}

	for _, ip := range managedIPs {
		// don't delete api server ip
		if *ip.Name == infraID+"-pip-v4" && m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
			continue
		}
		if _, ok := outboundIPMap[strings.ToLower(*ip.ID)]; !ok && strings.Contains(strings.ToLower(*ip.ID), strings.ToLower(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)) {
			ipName := stringutils.LastTokenByte(*ip.ID, '/')
			m.log.Infof("deleting managed public IP Address: %s", ipName)
			err := m.publicIPAddresses.DeleteAndWait(ctx, resourceGroupName, ipName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Returns the desired RP managed outbound publicIPAddresses.  Additional Managed Outbound IPs
// will be created as required to satisfy ManagedOutboundIP.Count.
func (m *manager) getDesiredOutboundIPs(ctx context.Context) ([]api.ResourceReference, error) {
	// Determine source of outbound IPs
	// TODO: add customer provided ip and ip prefixes
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		return m.reconcileDesiredManagedIPs(ctx)
	}
	return nil, nil
}

// Returns RP managed outbound ips to be added to the outbound rule.
// If the default outbound IP is present it will be added to ensure reuse of the ip when the
// api server is public.  If additional IPs are required they will be created.
func (m *manager) reconcileDesiredManagedIPs(ctx context.Context) ([]api.ResourceReference, error) {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	managedOBIPCount := m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count
	desiredIPAddresses := make([]api.ResourceReference, 0, managedOBIPCount)

	ipAddresses, err := m.getClusterManagedIPs(ctx)
	if err != nil {
		return nil, err
	}

	// create additional IPs if needed
	numToCreate := managedOBIPCount - len(ipAddresses)
	for i := 0; i < numToCreate; i++ {
		ipAddress, err := m.createPublicIPAddress(ctx)
		ipAddresses[*ipAddress.Name] = ipAddress
		if err != nil {
			return nil, err
		}
	}

	// ensure that when scaling managed ips down the default outbound IP is reused incase the api server visibility is public
	desiredCount := 0
	if defaultIP, ok := ipAddresses[infraID+"-pip-v4"]; ok {
		desiredIPAddresses = append(desiredIPAddresses, api.ResourceReference{ID: *defaultIP.ID})
		desiredCount += 1
		delete(ipAddresses, infraID+"-pip-v4")
	}

	for _, v := range ipAddresses {
		if desiredCount < managedOBIPCount {
			desiredIPAddresses = append(desiredIPAddresses, api.ResourceReference{ID: *v.ID})
			desiredCount += 1
		} else {
			break
		}
	}

	return desiredIPAddresses, nil
}

// Get all current managed IP Addresses in cluster resource group based on naming convention.
func (m *manager) getClusterManagedIPs(ctx context.Context) (map[string]mgmtnetwork.PublicIPAddress, error) {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	var ipAddresses = make(map[string]mgmtnetwork.PublicIPAddress)

	result, err := m.publicIPAddresses.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(result); i++ {
		// <infraID>-pip-v4 is not necessarily managed but is the default installed outbound IP
		if *result[i].Name == infraID+"-pip-v4" || strings.Contains(*result[i].Name, "-outbound-pip-v4") {
			ipAddresses[*result[i].Name] = result[i]
		}
	}

	return ipAddresses, err
}

func genManagedOutboundIPName() string {
	return uuid.DefaultGenerator.Generate() + "-outbound-pip-v4"
}

// Create a managed outbound IP Address.
func (m *manager) createPublicIPAddress(ctx context.Context) (mgmtnetwork.PublicIPAddress, error) {
	name := genManagedOutboundIPName()
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resourceID := fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, name)
	m.log.Infof("creating public IP Address: %s", name)
	publicIPAddress := mgmtnetwork.PublicIPAddress{
		Name:     &name,
		ID:       &resourceID,
		Location: &m.doc.OpenShiftCluster.Location,
		PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: mgmtnetwork.Static,
			PublicIPAddressVersion:   mgmtnetwork.IPv4,
		},
		Sku: &mgmtnetwork.PublicIPAddressSku{
			Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
		},
	}

	err := m.publicIPAddresses.CreateOrUpdateAndWait(ctx, resourceGroupName, name, publicIPAddress)
	if err != nil {
		return mgmtnetwork.PublicIPAddress{}, err
	}

	return publicIPAddress, nil
}

func getOutboundIPsFromLB(lb mgmtnetwork.LoadBalancer) []api.ResourceReference {
	var outboundIPs []api.ResourceReference
	fipConfigs := make(map[string]mgmtnetwork.FrontendIPConfiguration, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		fipConfigs[fipConfigID] = fipConfig
	}

	for _, obRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *obRule.Name == outboundRuleV4 {
			for i := 0; i < len(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations); i++ {
				id := *(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i].ID
				if fipConfig, ok := fipConfigs[id]; ok {
					outboundIPs = append(outboundIPs, api.ResourceReference{ID: *fipConfig.PublicIPAddress.ID})
				}
			}
		}
	}

	return outboundIPs
}

func (m *manager) patchEffectiveOutboundIPs(ctx context.Context, outboundIPs []api.ResourceReference) error {
	m.log.Info("patching effectiveOutboundIPs")
	effectiveOutboundIPs := make([]api.EffectiveOutboundIP, 0, len(outboundIPs))
	for _, obIP := range outboundIPs {
		effectiveOutboundIPs = append(effectiveOutboundIPs, api.EffectiveOutboundIP(obIP))
	}
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = effectiveOutboundIPs
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func newFrontendIPConfig(name string, id string, publicIPorIPPrefixID string) mgmtnetwork.FrontendIPConfiguration {
	// TODO: add check for publicIPorIPPrefixID
	return mgmtnetwork.FrontendIPConfiguration{
		Name: &name,
		ID:   &id,
		FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
			PublicIPAddress: &mgmtnetwork.PublicIPAddress{
				ID: &publicIPorIPPrefixID,
			},
		},
	}
}

func newOutboundRuleFrontendIPConfig(id string) mgmtnetwork.SubResource {
	return mgmtnetwork.SubResource{
		ID: &id,
	}
}

// Reports if two []api.ResourceReference are equal.
func areResourceRefsEqual(a, b []api.ResourceReference) bool {
	if len(a) != len(b) {
		return false
	}

	refsA := make([]string, 0, len(a))
	for _, ip := range a {
		refsA = append(refsA, ip.ID)
	}
	refsB := make([]string, 0, len(b))
	for _, ip := range b {
		refsB = append(refsB, ip.ID)
	}

	sort.Strings(refsA)
	sort.Strings(refsB)

	for i := 0; i < len(refsA); i++ {
		if !strings.EqualFold(refsA[i], refsB[i]) {
			return false
		}
	}

	return true
}
