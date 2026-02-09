package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const outboundRuleV4 = "outbound-rule-v4"

type deleteIPResult struct {
	name string
	err  error
}

type createIPResult struct {
	ip  sdknetwork.PublicIPAddress
	err error
}

// reconcileLoadBalancerProfile reconciles the outbound IPs of the public load balancer.
func (m *manager) reconcileLoadBalancerProfile(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType != api.OutboundTypeLoadbalancer || m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		return nil
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroupName, infraID, nil)
	if err != nil {
		return err
	}

	err = m.reconcileOutboundRuleV4IPs(ctx, lb.LoadBalancer)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) reconcileOutboundRuleV4IPs(ctx context.Context, lb sdknetwork.LoadBalancer) error {
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

func (m *manager) reconcileOutboundRuleV4IPsInner(ctx context.Context, lb sdknetwork.LoadBalancer) error {
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

	desiredOutboundIPs, err := m.reconcileOutboundIPs(ctx)
	if err != nil {
		return err
	}

	if areResourceRefsEqual(desiredOutboundIPs, originalOutboundIPs) {
		return nil
	}

	// rebuild outbound-rule-v4 frontend ip config with desired outbound ips
	removeOutboundIPsFromLB(lb)
	addOutboundIPsToLB(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, lb, desiredOutboundIPs)

	err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroupName, infraID, lb, nil)
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

// removeOutboundIPsFromLB removes outbound-rule-v4 IPs and corresponding frontendIPConfig from load balancer
func removeOutboundIPsFromLB(lb sdknetwork.LoadBalancer) {
	removeOutboundRuleV4FrontendIPConfig(lb)
	setOutboundRuleV4(lb, []*sdknetwork.SubResource{})
}

func removeOutboundRuleV4FrontendIPConfig(lb sdknetwork.LoadBalancer) {
	savedFIPConfig := make([]*sdknetwork.FrontendIPConfiguration, 0, len(lb.Properties.FrontendIPConfigurations))
	outboundRuleFrontendConfig := getOutboundRuleV4FIPConfigs(lb)

	for _, fipConfig := range lb.Properties.FrontendIPConfigurations {
		fipConfigID := *fipConfig.ID
		hasLBRules := fipConfig.Properties.LoadBalancingRules != nil
		if _, ok := outboundRuleFrontendConfig[fipConfigID]; ok && !hasLBRules {
			continue
		}
		savedFIPConfig = append(savedFIPConfig, fipConfig)
	}
	lb.Properties.FrontendIPConfigurations = savedFIPConfig
}

func getOutboundRuleV4FIPConfigs(lb sdknetwork.LoadBalancer) map[string]sdknetwork.SubResource {
	obRuleV4FIPConfigs := make(map[string]sdknetwork.SubResource)
	for _, obRule := range lb.Properties.OutboundRules {
		if *obRule.Name == outboundRuleV4 {
			for _, fipConfig := range obRule.Properties.FrontendIPConfigurations {
				fipConfigID := *fipConfig.ID
				obRuleV4FIPConfigs[fipConfigID] = *fipConfig
			}
			break
		}
	}
	return obRuleV4FIPConfigs
}

// getFrontendIPConfigs returns a map of Frontend IP Configurations from the given load balancer
func getFrontendIPConfigs(lb sdknetwork.LoadBalancer) map[string]sdknetwork.FrontendIPConfiguration {
	frontendIPConfigs := make(map[string]sdknetwork.FrontendIPConfiguration, len(lb.Properties.FrontendIPConfigurations))

	for _, fipConfig := range lb.Properties.FrontendIPConfigurations {
		fipConfigID := *fipConfig.ID
		fipConfigIPAddressID := *fipConfig.Properties.PublicIPAddress.ID
		frontendIPConfigs[fipConfigID] = *fipConfig
		frontendIPConfigs[fipConfigIPAddressID] = *fipConfig
	}

	return frontendIPConfigs
}

// addOutboundIPsToLB adds IPs or IPPrefixes to the load balancer outbound rule "outbound-rule-v4".
func addOutboundIPsToLB(resourceGroupID string, lb sdknetwork.LoadBalancer, obIPsOrIPPrefixes []api.ResourceReference) {
	frontendIPConfigs := getFrontendIPConfigs(lb)
	var outboundRuleV4FrontendIPConfig []*sdknetwork.SubResource

	// add IP Addresses to frontendConfig
	for _, obIPOrIPPrefix := range obIPsOrIPPrefixes {
		// check if the frontend config exists in the map to avoid duplicate entries
		if _, ok := frontendIPConfigs[obIPOrIPPrefix.ID]; !ok {
			frontendIPConfigName := stringutils.LastTokenByte(obIPOrIPPrefix.ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, newFrontendIPConfig(frontendIPConfigName, frontendConfigID, obIPOrIPPrefix.ID))
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(frontendConfigID))
		} else {
			// frontendIPConfig already exists and just needs to be added to the outbound rule
			frontendConfig := frontendIPConfigs[obIPOrIPPrefix.ID]
			outboundRuleV4FrontendIPConfig = append(outboundRuleV4FrontendIPConfig, newOutboundRuleFrontendIPConfig(*frontendConfig.ID))
		}
	}

	setOutboundRuleV4(lb, outboundRuleV4FrontendIPConfig)
}

func setOutboundRuleV4(lb sdknetwork.LoadBalancer, outboundRuleV4FrontendIPConfig []*sdknetwork.SubResource) {
	for _, outboundRule := range lb.Properties.OutboundRules {
		if *outboundRule.Name == outboundRuleV4 {
			outboundRule.Properties.FrontendIPConfigurations = outboundRuleV4FrontendIPConfig
			break
		}
	}
}

// deleteUnusedManagedIPs all managed outbound IPs that are not in use by the load balancer.
// The default outbound ip is saved if the api server is public.
func (m *manager) deleteUnusedManagedIPs(ctx context.Context) error {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	unusedManagedIPs, err := m.getUnusedManagedIPs(ctx)
	if err != nil {
		return err
	}

	ch := make(chan deleteIPResult)
	defer close(ch)
	var cleanupErrors []string

	for _, id := range unusedManagedIPs {
		ipName := stringutils.LastTokenByte(id, '/')
		go m.deleteIPAddress(ctx, resourceGroupName, ipName, ch)
	}

	for range unusedManagedIPs {
		result := <-ch
		if result.err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("deletion of unused managed ip %s failed with error: %v", result.name, result.err))
		}
	}

	if cleanupErrors != nil {
		return fmt.Errorf("failed to cleanup unused managed ips\n%s", strings.Join(cleanupErrors, "\n"))
	}

	return nil
}

func (m *manager) deleteIPAddress(ctx context.Context, resourceGroupName string, ipName string, ch chan<- deleteIPResult) {
	m.log.Infof("deleting managed public IP Address: %s", ipName)
	err := m.armPublicIPAddresses.DeleteAndWait(ctx, resourceGroupName, ipName, nil)
	ch <- deleteIPResult{
		name: ipName,
		err:  err,
	}
}

func (m *manager) getUnusedManagedIPs(ctx context.Context) ([]string, error) {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	managedIPs, err := m.getClusterManagedIPs(ctx)
	if err != nil {
		return nil, err
	}

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroupName, infraID, nil)
	if err != nil {
		return nil, err
	}

	outboundIPs := getOutboundIPsFromLB(lb.LoadBalancer)
	outboundIPMap := make(map[string]api.ResourceReference, len(outboundIPs))
	for i := 0; i < len(outboundIPs); i++ {
		outboundIPMap[strings.ToLower(outboundIPs[i].ID)] = outboundIPs[i]
	}
	var unusedManagedIPs []string
	for _, ip := range managedIPs {
		// don't delete api server ip
		if *ip.Name == infraID+"-pip-v4" && m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
			continue
		}
		if _, ok := outboundIPMap[strings.ToLower(*ip.ID)]; !ok && strings.Contains(strings.ToLower(*ip.ID), strings.ToLower(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)) {
			unusedManagedIPs = append(unusedManagedIPs, *ip.ID)
		}
	}
	return unusedManagedIPs, nil
}

// reconcileOutboundIPs returns the desired RP managed outbound publicIPAddresses.
// Additional Managed Outbound IPs will be created as required to satisfy ManagedOutboundIP.Count.
func (m *manager) reconcileOutboundIPs(ctx context.Context) ([]api.ResourceReference, error) {
	// Determine source of outbound IPs
	// TODO: add customer provided ip and ip prefixes
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		return m.reconcileDesiredManagedIPs(ctx)
	}
	return nil, nil
}

// reconcileDesiredManagedIPs returns RP managed outbound ips to be added to the outbound rule.
// If the default outbound IP is present it will be added to ensure reuse of the ip when the
// api server is public.  If additional IPs are required they will be created.
func (m *manager) reconcileDesiredManagedIPs(ctx context.Context) ([]api.ResourceReference, error) {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	managedOBIPCount := m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count

	ipAddresses, err := m.getClusterManagedIPs(ctx)
	if err != nil {
		return nil, err
	}

	numToCreate := managedOBIPCount - len(ipAddresses)

	if numToCreate > 0 {
		err = m.createPublicIPAddresses(ctx, ipAddresses, numToCreate)
		if err != nil {
			return nil, err
		}
	}

	desiredIPAddresses := getDesiredOutboundIPs(managedOBIPCount, ipAddresses, infraID)
	return desiredIPAddresses, nil
}

// getDesiredOutboundIPs returns the desired outbound IPs to be used by the load balancer.
func getDesiredOutboundIPs(managedOBIPCount int, ipAddresses map[string]sdknetwork.PublicIPAddress, infraID string) []api.ResourceReference {
	desiredIPAddresses := make([]api.ResourceReference, 0, managedOBIPCount)
	// ensure that when scaling managed ips down the default outbound IP is reused in case the api server visibility is public
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
	return desiredIPAddresses
}

// createPublicIPAddresses creates managed outbound IP Addresses.
func (m *manager) createPublicIPAddresses(ctx context.Context, ipAddresses map[string]sdknetwork.PublicIPAddress, numToCreate int) error {
	ch := make(chan createIPResult)
	defer close(ch)
	var errResults []string
	// create additional IPs if needed
	for i := 0; i < numToCreate; i++ {
		go m.createPublicIPAddress(ctx, ch)
	}

	for i := 0; i < numToCreate; i++ {
		result := <-ch
		if result.err != nil {
			errResults = append(errResults, fmt.Sprintf("creation of ip address %s failed with error: %s", *result.ip.Name, result.err.Error()))
		} else {
			ipAddresses[*result.ip.Name] = result.ip
		}
	}

	if len(errResults) > 0 {
		return fmt.Errorf("failed to create required IPs\n%s", strings.Join(errResults, "\n"))
	}
	return nil
}

// getClusterManagedIPs gets all current managed IP Addresses in cluster resource group based on naming convention.
func (m *manager) getClusterManagedIPs(ctx context.Context) (map[string]sdknetwork.PublicIPAddress, error) {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	ipAddresses := make(map[string]sdknetwork.PublicIPAddress)

	result, err := m.armPublicIPAddresses.List(ctx, resourceGroupName, nil)
	if err != nil {
		return nil, err
	}

	for _, ip := range result {
		// <infraID>-pip-v4 is the default installed outbound IP
		if *ip.Name == infraID+"-pip-v4" || strings.Contains(*ip.Name, "-outbound-pip-v4") {
			ipAddresses[*ip.Name] = *ip
		}
	}

	return ipAddresses, err
}

func genManagedOutboundIPName() string {
	return uuid.DefaultGenerator.Generate() + "-outbound-pip-v4"
}

// createPublicIPAddress creates a managed outbound IP Address.
func (m *manager) createPublicIPAddress(ctx context.Context, ch chan<- createIPResult) {
	name := genManagedOutboundIPName()
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resourceID := fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, name)
	m.log.Infof("creating public IP Address: %s", name)
	publicIPAddress := newPublicIPAddress(name, resourceID, m.doc.OpenShiftCluster.Location)

	err := m.armPublicIPAddresses.CreateOrUpdateAndWait(ctx, resourceGroupName, name, publicIPAddress, nil)
	ch <- createIPResult{
		ip:  publicIPAddress,
		err: err,
	}
}

// getOutboundIPsFromLB returns the public IP addresses used by the load balancer outbound rule "outbound-rule-v4".
func getOutboundIPsFromLB(lb sdknetwork.LoadBalancer) []api.ResourceReference {
	var outboundIPs []api.ResourceReference
	fipConfigs := getFrontendIPConfigs(lb)

	for _, obRule := range lb.Properties.OutboundRules {
		if *obRule.Name == outboundRuleV4 {
			for _, obFipConfig := range obRule.Properties.FrontendIPConfigurations {
				id := *obFipConfig.ID
				if fipConfig, ok := fipConfigs[id]; ok {
					outboundIPs = append(outboundIPs, api.ResourceReference{ID: *fipConfig.Properties.PublicIPAddress.ID})
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

func newPublicIPAddress(name, resourceID, location string) sdknetwork.PublicIPAddress {
	return sdknetwork.PublicIPAddress{
		Name:     &name,
		ID:       &resourceID,
		Location: &location,
		Properties: &sdknetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: pointerutils.ToPtr(sdknetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:   pointerutils.ToPtr(sdknetwork.IPVersionIPv4),
		},
		SKU: &sdknetwork.PublicIPAddressSKU{
			Name: pointerutils.ToPtr(sdknetwork.PublicIPAddressSKUNameStandard),
		},
	}
}

func newFrontendIPConfig(name string, id string, publicIPorIPPrefixID string) *sdknetwork.FrontendIPConfiguration {
	// TODO: add check for publicIPorIPPrefixID
	return &sdknetwork.FrontendIPConfiguration{
		Name: &name,
		ID:   &id,
		Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
			PublicIPAddress: &sdknetwork.PublicIPAddress{
				ID: &publicIPorIPPrefixID,
			},
		},
	}
}

func newOutboundRuleFrontendIPConfig(id string) *sdknetwork.SubResource {
	return &sdknetwork.SubResource{
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
