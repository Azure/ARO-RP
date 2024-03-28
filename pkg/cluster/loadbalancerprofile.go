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
	"github.com/Azure/ARO-RP/pkg/util/loadbalancer"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

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
	originalOutboundIPs := loadbalancer.GetOutboundIPsFromLB(lb)

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
	loadbalancer.RemoveOutboundIPsFromLB(lb)
	loadbalancer.AddOutboundIPsToLB(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, lb, desiredOutboundIPs)

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

type deleteResult struct {
	name string
	err  error
}

// Delete all managed outbound IPs that are not in use by the load balancer.
// The default outbound ip is saved if the api server is public.
func (m *manager) deleteUnusedManagedIPs(ctx context.Context) error {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	ipsToDelete, err := m.getManagedIPsToDelete(ctx)
	if err != nil {
		return err
	}

	ch := make(chan deleteResult)
	defer close(ch)
	var cleanupErrors []string

	for _, id := range ipsToDelete {
		ipName := stringutils.LastTokenByte(id, '/')
		go m.deleteIPAddress(ctx, resourceGroupName, ipName, ch)
	}

	for range ipsToDelete {
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

func (m *manager) deleteIPAddress(ctx context.Context, resourceGroupName string, ipName string, ch chan<- deleteResult) {
	m.log.Infof("deleting managed public IP Address: %s", ipName)
	err := m.publicIPAddresses.DeleteAndWait(ctx, resourceGroupName, ipName)
	ch <- deleteResult{
		name: ipName,
		err:  err,
	}
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

	outboundIPs := loadbalancer.GetOutboundIPsFromLB(lb)
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
func (m *manager) reconcileOutboundIPs(ctx context.Context) ([]api.ResourceReference, error) {
	// Determine source of outbound IPs
	// TODO: add customer provided ip and ip prefixes
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		return m.reconcileDesiredManagedIPs(ctx)
	}
	return nil, nil
}

type createIPResult struct {
	ip  mgmtnetwork.PublicIPAddress
	err error
}

// Returns RP managed outbound ips to be added to the outbound rule.
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

func getDesiredOutboundIPs(managedOBIPCount int, ipAddresses map[string]mgmtnetwork.PublicIPAddress, infraID string) []api.ResourceReference {
	desiredIPAddresses := make([]api.ResourceReference, 0, managedOBIPCount)
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
	return desiredIPAddresses
}

func (m *manager) createPublicIPAddresses(ctx context.Context, ipAddresses map[string]mgmtnetwork.PublicIPAddress, numToCreate int) error {
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
func (m *manager) createPublicIPAddress(ctx context.Context, ch chan<- createIPResult) {
	name := genManagedOutboundIPName()
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resourceID := fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, name)
	m.log.Infof("creating public IP Address: %s", name)
	publicIPAddress := newPublicIPAddress(name, resourceID, m.doc.OpenShiftCluster.Location)

	err := m.publicIPAddresses.CreateOrUpdateAndWait(ctx, resourceGroupName, name, publicIPAddress)
	ch <- createIPResult{
		ip:  publicIPAddress,
		err: err,
	}
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

func newPublicIPAddress(name, resourceID, location string) mgmtnetwork.PublicIPAddress {
	return mgmtnetwork.PublicIPAddress{
		Name:     &name,
		ID:       &resourceID,
		Location: &location,
		PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: mgmtnetwork.Static,
			PublicIPAddressVersion:   mgmtnetwork.IPv4,
		},
		Sku: &mgmtnetwork.PublicIPAddressSku{
			Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
		},
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
