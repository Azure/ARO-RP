package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const outboundRuleV4 = "outbound-rule-v4"

func (m *manager) reconcileLoadBalancerProfile(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType != api.OutboundTypeLoadbalancer || m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
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

func (m *manager) reconcileOutboundRuleV4IPs(ctx context.Context, lb mgmtnetwork.LoadBalancer) error {
	err := m.reconcileOutboundRuleV4IPsInner(ctx, lb)

	cleanupError := m.deleteUnusedManagedIPs(ctx)
	if cleanupError != nil {
		if err == nil {
			return cleanupError
		}
		return fmt.Errorf("multiple errors occurred while updating outbound-rule-v4\n%v\n%v", err, cleanupError)
	}

	return err
}

func (m *manager) reconcileOutboundRuleV4IPsInner(ctx context.Context, lb mgmtnetwork.LoadBalancer) error {
	m.log.Info("reconciling outbound-rule-v4")

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	originalOutboundIPs := getOutboundIPsFromLB(lb)

	if needsEffectiveOutboundIPsPatched(m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs, originalOutboundIPs) {
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

// Remove outbound-rule-v4 IPs and corresponding frontendIPConfig from load balancer
func removeOutboundIPsFromLB(lb mgmtnetwork.LoadBalancer) {
	removeOutboundRuleV4FrontendIPConfig(lb)
	setOutboundRuleV4(lb, []mgmtnetwork.SubResource{})
}

func removeOutboundRuleV4FrontendIPConfig(lb mgmtnetwork.LoadBalancer) {
	var savedFIPConfig = make([]mgmtnetwork.FrontendIPConfiguration, 0, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))
	var outboundRuleFrontendConfig = getOutboundRuleV4FIPConfigs(lb)

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		hasLBRules := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].LoadBalancingRules != nil
		if _, ok := outboundRuleFrontendConfig[fipConfigID]; ok && !hasLBRules {
			continue
		}
		savedFIPConfig = append(savedFIPConfig, fipConfig)
	}
	lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &savedFIPConfig
}

func getOutboundRuleV4FIPConfigs(lb mgmtnetwork.LoadBalancer) map[string]mgmtnetwork.SubResource {
	var obRuleV4FIPConfigs = make(map[string]mgmtnetwork.SubResource)
	for _, obRule := range *lb.LoadBalancerPropertiesFormat.OutboundRules {
		if *obRule.Name == outboundRuleV4 {
			for i := 0; i < len(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations); i++ {
				fipConfigID := *(*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i].ID
				fipConfig := (*obRule.OutboundRulePropertiesFormat.FrontendIPConfigurations)[i]
				obRuleV4FIPConfigs[fipConfigID] = fipConfig
			}
			break
		}
	}
	return obRuleV4FIPConfigs
}

// Returns a map of Frontend IP Configurations.  Frontend IP Configurations can be looked up by Public IP Address ID or Frontend IP Configuration ID
func getFrontendIPConfigs(lb mgmtnetwork.LoadBalancer) map[string]mgmtnetwork.FrontendIPConfiguration {
	var frontendIPConfigs = make(map[string]mgmtnetwork.FrontendIPConfiguration, len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations))

	for i := 0; i < len(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations); i++ {
		fipConfigID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].ID
		fipConfigIPAddressID := *(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i].FrontendIPConfigurationPropertiesFormat.PublicIPAddress.ID
		fipConfig := (*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations)[i]
		frontendIPConfigs[fipConfigID] = fipConfig
		frontendIPConfigs[fipConfigIPAddressID] = fipConfig
	}

	return frontendIPConfigs
}

// Adds IPs or IPPrefixes to the load balancer outbound rule "outbound-rule-v4".
func addOutboundIPsToLB(resourceGroupID string, lb mgmtnetwork.LoadBalancer, obIPsOrIPPrefixes []api.ResourceReference) {
	frontendIPConfigs := getFrontendIPConfigs(lb)
	outboundRuleV4FrontendIPConfig := []mgmtnetwork.SubResource{}

	// add IP Addresses to frontendConfig
	for _, obIPOrIPPrefix := range obIPsOrIPPrefixes {
		// check if the frontend config exists in the map to avoid duplicate entries
		if _, ok := frontendIPConfigs[obIPOrIPPrefix.ID]; !ok {
			frontendIPConfigName := stringutils.LastTokenByte(obIPOrIPPrefix.ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations, newFrontendIPConfig(frontendIPConfigName, frontendConfigID, obIPOrIPPrefix.ID))
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(frontendConfigID))
		} else {
			// frontendIPConfig already exists and just needs to be added to the outbound rule
			frontendConfig := frontendIPConfigs[obIPOrIPPrefix.ID]
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(*frontendConfig.ID))
		}
	}

	setOutboundRuleV4(lb, outboundRuleV4FrontendIPConfig)
}

func setOutboundRuleV4(lb mgmtnetwork.LoadBalancer, outboundRuleV4FrontendIPConfig []mgmtnetwork.SubResource) {
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

	ipsToDelete, err := m.getManagedIPsToDelete(ctx)
	if err != nil {
		return err
	}
	var cleanupErrors []string
	for _, id := range ipsToDelete {
		ipName := stringutils.LastTokenByte(id, '/')
		m.log.Infof("deleting managed public IP Address: %s", ipName)
		err := m.publicIPAddresses.DeleteAndWait(ctx, resourceGroupName, ipName)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("deletion of unused managed ip %s failed with error: %v", ipName, err))
		}
	}
	if cleanupErrors != nil {
		return fmt.Errorf("failed to cleanup unused managed ips\n%s", strings.Join(cleanupErrors, "\n"))
	}

	return nil
}

func (m *manager) getManagedIPsToDelete(ctx context.Context) ([]string, error) {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	managedIPs, err := m.getClusterManagedIPs(ctx)
	if err != nil {
		return nil, err
	}

	lb, err := m.loadBalancers.Get(ctx, resourceGroupName, infraID, "")
	if err != nil {
		return nil, err
	}

	outboundIPs := getOutboundIPsFromLB(lb)
	outboundIPMap := make(map[string]api.ResourceReference, len(outboundIPs))
	for i := 0; i < len(outboundIPs); i++ {
		outboundIPMap[strings.ToLower(outboundIPs[i].ID)] = outboundIPs[i]
	}
	var ipsToDelete []string
	for _, ip := range managedIPs {
		// don't delete api server ip
		if *ip.Name == infraID+"-pip-v4" && m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
			continue
		}
		if _, ok := outboundIPMap[strings.ToLower(*ip.ID)]; !ok && strings.Contains(strings.ToLower(*ip.ID), strings.ToLower(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)) {
			ipsToDelete = append(ipsToDelete, *ip.ID)
		}
	}
	return ipsToDelete, nil
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
		if err != nil {
			return nil, err
		}
		ipAddresses[*ipAddress.Name] = ipAddress
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
		// <infraID>-pip-v4 is the default installed outbound IP
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
	fipConfigs := getFrontendIPConfigs(lb)

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

func needsEffectiveOutboundIPsPatched(cosmosEffectiveOutboundIPs []api.EffectiveOutboundIP, lbEffectiveIPs []api.ResourceReference) bool {
	if cosmosEffectiveOutboundIPs == nil {
		return true
	}
	effectiveIPResources := make([]api.ResourceReference, 0, len(cosmosEffectiveOutboundIPs))
	for _, ip := range cosmosEffectiveOutboundIPs {
		effectiveIPResources = append(effectiveIPResources, api.ResourceReference(ip))
	}
	return !areResourceRefsEqual(effectiveIPResources, lbEffectiveIPs)
}

// Reports if two []api.ResourceReference are equal.
func areResourceRefsEqual(a, b []api.ResourceReference) bool {
	if len(a) != len(b) {
		return false
	}

	refsA := make([]string, 0, len(a))
	for _, ip := range a {
		refsA = append(refsA, strings.ToLower(ip.ID))
	}
	refsB := make([]string, 0, len(b))
	for _, ip := range b {
		refsB = append(refsB, strings.ToLower(ip.ID))
	}

	sort.Strings(refsA)
	sort.Strings(refsB)

	return reflect.DeepEqual(refsA, refsB)
}
