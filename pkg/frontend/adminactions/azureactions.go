package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	sdkcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// AzureActions contains those actions which rely solely on Azure clients, not using any k8s clients
type AzureActions interface {
	GroupResourceList(ctx context.Context) ([]mgmtfeatures.GenericResourceExpanded, error)
	ResourcesList(ctx context.Context, resources []mgmtfeatures.GenericResourceExpanded, writer io.WriteCloser) error
	WriteToStream(ctx context.Context, writer io.WriteCloser) error
	NICReconcileFailedState(ctx context.Context, nicName string) error
	VMRedeployAndWait(ctx context.Context, vmName string) error
	VMStartAndWait(ctx context.Context, vmName string) error
	VMStopAndWait(ctx context.Context, vmName string, deallocateVM bool) error
	VMSizeList(ctx context.Context) ([]string, error)
	VMGetSKUs(ctx context.Context, vmSizes []string) (map[string]*sdkcompute.ResourceSKU, error)
	VMResize(ctx context.Context, vmName string, vmSize string) error
	ResourceGroupHasVM(ctx context.Context, vmName string) (bool, error)
	VMSerialConsole(ctx context.Context, log *logrus.Entry, vmName string, target io.Writer) error
	ResourceDeleteAndWait(ctx context.Context, resourceID string) error
	GetEffectiveRouteTable(ctx context.Context, nicName string) ([]byte, error)
	GetVirtualMachine(ctx context.Context, resourceGroupName string, VMName string, expand mgmtcompute.InstanceViewTypes) (result mgmtcompute.VirtualMachine, err error)
	GetNetworkInterfaceSSHInfo(ctx context.Context, resourceGroupName, networkInterfaceName string) (*NetworkInterfaceSSHInfo, error)
	GetSSHRouteStatus(ctx context.Context, resourceGroupName, clusterResourceGroupID, loadBalancerName, backendPoolID, selectedNetworkInterfaceID, frontendIP string, frontendPort int32) (*SSHRouteStatus, error)
	CreateCRG(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error)
	CreateCapacityReservation(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error
	DeleteCRG(ctx context.Context, clusterRG, crgName string) error
	DeleteCapacityReservation(ctx context.Context, clusterRG, crgName, zone string) error
	ListComputeUsage(ctx context.Context, location string) ([]mgmtcompute.Usage, error)
}

type SSHRouteStatus struct {
	BackendPoolNetworkInterfaceIDs []string
	LoadBalancingRuleCount         int
}

type NetworkInterfaceSSHInfo struct {
	ID             string
	BackendPoolIDs []string
	SubnetIDs      []string
}

type azureActions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	networkInterfaces  armnetwork.InterfacesClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	loadBalancers      armnetwork.LoadBalancersClient
	resources          features.ResourcesClient
	resourceSkus       armcompute.ResourceSKUsClient
	routeTables        armnetwork.RouteTablesClient
	securityGroups     armnetwork.SecurityGroupsClient
	storageAccounts    storage.AccountsClient
	virtualMachines    compute.VirtualMachinesClient
	computeUsage       compute.UsageClient
	virtualNetworks    armnetwork.VirtualNetworksClient

	capacityReservationGroups armcompute.CapacityReservationGroupsClient
	capacityReservations      armcompute.CapacityReservationsClient
}

// NewAzureActions returns an azureActions
func NewAzureActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument,
) (AzureActions, error) {
	fpAuth, err := env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, nil,
		env.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	credential, err := env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID, nil)
	if err != nil {
		return nil, err
	}

	options := env.Environment().ArmClientOptions()

	networkInterfaces, err := armnetwork.NewInterfacesClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	loadBalancers, err := armnetwork.NewLoadBalancersClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	routeTables, err := armnetwork.NewRouteTablesClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	securityGroups, err := armnetwork.NewSecurityGroupsClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	virtualNetworks, err := armnetwork.NewVirtualNetworksClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	armResourceSKUsClient, err := armcompute.NewResourceSKUsClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	armCapacityReservationGroups, err := armcompute.NewCapacityReservationGroupsClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	armCapacityReservations, err := armcompute.NewCapacityReservationsClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	return &azureActions{
		log: log,
		env: env,
		oc:  oc,

		networkInterfaces:  networkInterfaces,
		diskEncryptionSets: compute.NewDiskEncryptionSetsClientWithAROEnvironment(env.Environment(), subscriptionDoc.ID, fpAuth),
		loadBalancers:      loadBalancers,
		resources:          features.NewResourcesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		resourceSkus:       armResourceSKUsClient,
		routeTables:        routeTables,
		securityGroups:     securityGroups,
		storageAccounts:    storage.NewAccountsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualMachines:    compute.NewVirtualMachinesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		computeUsage:       compute.NewUsageClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualNetworks:    virtualNetworks,

		capacityReservationGroups: armCapacityReservationGroups,
		capacityReservations:      armCapacityReservations,
	}, nil
}

func (a *azureActions) VMRedeployAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	vm, err := a.virtualMachines.Get(ctx, clusterRGName, vmName, mgmtcompute.InstanceView)
	if err != nil {
		return err
	}
	if vmDisk := vm.StorageProfile.OsDisk; vmDisk != nil && vmDisk.DiffDiskSettings != nil &&
		vmDisk.Caching == "ReadOnly" && vmDisk.DiffDiskSettings.Option == "Local" && vmDisk.DiffDiskSettings.Placement == "CacheDisk" {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", fmt.Sprintf("VM '%s' has an Ephemeral Disk OS and cannot be redeployed.", vmName))
	}
	return a.virtualMachines.RedeployAndWait(ctx, clusterRGName, vmName)
}

func (a *azureActions) VMStartAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.StartAndWait(ctx, clusterRGName, vmName)
}

func (a *azureActions) VMStopAndWait(ctx context.Context, vmName string, deallocateVM bool) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.StopAndWait(ctx, clusterRGName, vmName, deallocateVM)
}

func (a *azureActions) VMGetSKUs(ctx context.Context, vmSizes []string) (map[string]*sdkcompute.ResourceSKU, error) {
	return computeskus.SelectVMSkusInCurrentRegion(ctx, a.resourceSkus, a.env.Location(), vmSizes)
}

func (a *azureActions) VMSizeList(ctx context.Context) ([]string, error) {
	return computeskus.ListUnrestrictedVMSkusInCurrentRegion(ctx, a.resourceSkus, a.env.Location())
}

func (a *azureActions) VMResize(ctx context.Context, vmName string, size string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	vm, err := a.virtualMachines.Get(ctx, clusterRGName, vmName, mgmtcompute.InstanceView)
	if err != nil {
		return err
	}

	vm.HardwareProfile.VMSize = mgmtcompute.VirtualMachineSizeTypes(size)
	return a.virtualMachines.CreateOrUpdateAndWait(ctx, clusterRGName, vmName, vm)
}

func (a *azureActions) ResourceGroupHasVM(ctx context.Context, vmName string) (bool, error) {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	vmList, err := a.virtualMachines.List(ctx, clusterRGName)
	if err != nil {
		return false, err
	}

	for _, vm := range vmList {
		if vm.Name != nil && *vm.Name == vmName {
			return true, nil
		}
	}

	return false, nil
}

func (a *azureActions) ListComputeUsage(ctx context.Context, location string) ([]mgmtcompute.Usage, error) {
	return a.computeUsage.List(ctx, location)
}

func (a *azureActions) GetEffectiveRouteTable(ctx context.Context, nicName string) ([]byte, error) {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	result, err := a.networkInterfaces.GetEffectiveRouteTableAndWait(ctx, clusterRGName, nicName, nil)
	if err != nil {
		return nil, err
	}

	jsonData, err := result.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (a *azureActions) GetVirtualMachine(ctx context.Context, resourceGroupName string, VMName string, expand mgmtcompute.InstanceViewTypes) (result mgmtcompute.VirtualMachine, err error) {
	return a.virtualMachines.Get(ctx, resourceGroupName, VMName, expand)
}

func (a *azureActions) GetNetworkInterfaceSSHInfo(ctx context.Context, resourceGroupName, networkInterfaceName string) (*NetworkInterfaceSSHInfo, error) {
	resp, err := a.networkInterfaces.Get(ctx, resourceGroupName, networkInterfaceName, nil)
	if err != nil {
		return nil, err
	}

	info := &NetworkInterfaceSSHInfo{
		BackendPoolIDs: []string{},
		SubnetIDs:      []string{},
	}
	if resp.ID == nil {
		return nil, fmt.Errorf("network interface %q has nil ID", networkInterfaceName)
	}
	info.ID = *resp.ID
	if resp.Properties == nil {
		return nil, fmt.Errorf("network interface %q has nil properties", networkInterfaceName)
	}
	for ipConfigIndex, ipConfig := range resp.Properties.IPConfigurations {
		if ipConfig == nil {
			return nil, fmt.Errorf("network interface %q has nil IP configuration at index %d", networkInterfaceName, ipConfigIndex)
		}
		if ipConfig.Properties == nil {
			return nil, fmt.Errorf("network interface %q IP configuration at index %d has nil properties", networkInterfaceName, ipConfigIndex)
		}
		if ipConfig.Properties.Subnet == nil || ipConfig.Properties.Subnet.ID == nil {
			return nil, fmt.Errorf("network interface %q IP configuration at index %d has no subnet ID", networkInterfaceName, ipConfigIndex)
		}
		info.SubnetIDs = append(info.SubnetIDs, *ipConfig.Properties.Subnet.ID)
		for poolIndex, pool := range ipConfig.Properties.LoadBalancerBackendAddressPools {
			if pool == nil {
				return nil, fmt.Errorf("network interface %q IP configuration at index %d has nil backend pool at index %d", networkInterfaceName, ipConfigIndex, poolIndex)
			}
			if pool.ID == nil {
				return nil, fmt.Errorf("network interface %q IP configuration at index %d backend pool at index %d has nil ID", networkInterfaceName, ipConfigIndex, poolIndex)
			}
			info.BackendPoolIDs = append(info.BackendPoolIDs, *pool.ID)
		}
	}
	return info, nil
}

func (a *azureActions) GetSSHRouteStatus(ctx context.Context, resourceGroupName, clusterResourceGroupID, loadBalancerName, backendPoolID, selectedNetworkInterfaceID, frontendIP string, frontendPort int32) (*SSHRouteStatus, error) {
	lb, err := a.loadBalancers.Get(ctx, resourceGroupName, loadBalancerName, nil)
	if err != nil {
		return nil, err
	}
	if lb.Properties == nil {
		return nil, fmt.Errorf("load balancer %q has nil properties", loadBalancerName)
	}
	frontendIDs := map[string]struct{}{}
	for frontendIndex, frontend := range lb.Properties.FrontendIPConfigurations {
		if frontend == nil || frontend.ID == nil || frontend.Properties == nil || frontend.Properties.PrivateIPAddress == nil {
			continue
		}
		candidateFrontendIP := net.ParseIP(*frontend.Properties.PrivateIPAddress)
		expectedIP := net.ParseIP(frontendIP)
		if candidateFrontendIP == nil || expectedIP == nil {
			return nil, fmt.Errorf("load balancer %q frontend at index %d has invalid private IP", loadBalancerName, frontendIndex)
		}
		if candidateFrontendIP.Equal(expectedIP) {
			frontendIDs[strings.ToLower(*frontend.ID)] = struct{}{}
		}
	}

	poolNICIDs := map[string]string{}
	poolFound := false
	clusterNICPrefix := strings.ToLower(strings.TrimSuffix(clusterResourceGroupID, "/") + "/providers/Microsoft.Network/networkInterfaces/")
	for poolIndex, pool := range lb.Properties.BackendAddressPools {
		if pool == nil || pool.ID == nil || !strings.EqualFold(*pool.ID, backendPoolID) {
			continue
		}
		if poolFound {
			return nil, fmt.Errorf("load balancer %q has duplicate backend pool %q", loadBalancerName, backendPoolID)
		}
		poolFound = true
		if pool.Properties == nil {
			return nil, fmt.Errorf("load balancer %q backend pool at index %d has nil properties", loadBalancerName, poolIndex)
		}
		if len(pool.Properties.LoadBalancerBackendAddresses) != 0 {
			return nil, fmt.Errorf("load balancer %q backend pool %q contains IP-based backend members", loadBalancerName, backendPoolID)
		}
		for ipConfigIndex, ipConfig := range pool.Properties.BackendIPConfigurations {
			if ipConfig == nil || ipConfig.ID == nil {
				return nil, fmt.Errorf("load balancer %q backend pool %q has nil IP configuration ID at index %d", loadBalancerName, backendPoolID, ipConfigIndex)
			}
			lowerID := strings.ToLower(*ipConfig.ID)
			marker := "/ipconfigurations/"
			markerIndex := strings.LastIndex(lowerID, marker)
			if markerIndex < 0 {
				return nil, fmt.Errorf("load balancer %q backend pool %q has invalid IP configuration ID %q", loadBalancerName, backendPoolID, *ipConfig.ID)
			}
			nicID := (*ipConfig.ID)[:markerIndex]
			if !strings.HasPrefix(strings.ToLower(nicID)+"/", clusterNICPrefix) {
				return nil, fmt.Errorf("load balancer %q backend pool %q references NIC %q outside the cluster resource group", loadBalancerName, backendPoolID, nicID)
			}
			poolNICIDs[strings.ToLower(nicID)] = nicID
		}
	}
	if !poolFound {
		return nil, fmt.Errorf("load balancer %q does not contain backend pool %q", loadBalancerName, backendPoolID)
	}

	matchingRules := 0
	for ruleIndex, rule := range lb.Properties.LoadBalancingRules {
		if rule == nil || rule.Properties == nil || rule.Properties.BackendAddressPool == nil || rule.Properties.BackendAddressPool.ID == nil {
			continue
		}
		if !strings.EqualFold(*rule.Properties.BackendAddressPool.ID, backendPoolID) {
			continue
		}
		if rule.Properties.FrontendIPConfiguration == nil || rule.Properties.FrontendIPConfiguration.ID == nil || rule.Properties.FrontendPort == nil || rule.Properties.BackendPort == nil || rule.Properties.Protocol == nil {
			return nil, fmt.Errorf("load balancer %q SSH rule at index %d is incomplete", loadBalancerName, ruleIndex)
		}
		_, frontendMatches := frontendIDs[strings.ToLower(*rule.Properties.FrontendIPConfiguration.ID)]
		if frontendMatches && *rule.Properties.FrontendPort == frontendPort && *rule.Properties.BackendPort == 22 && *rule.Properties.Protocol == sdknetwork.TransportProtocolTCP {
			matchingRules++
		}
	}

	status := &SSHRouteStatus{LoadBalancingRuleCount: matchingRules}
	for _, interfaceID := range poolNICIDs {
		status.BackendPoolNetworkInterfaceIDs = append(status.BackendPoolNetworkInterfaceIDs, interfaceID)
	}
	if _, ok := poolNICIDs[strings.ToLower(selectedNetworkInterfaceID)]; !ok {
		return nil, fmt.Errorf("load balancer %q backend pool %q does not contain selected NIC %q", loadBalancerName, backendPoolID, selectedNetworkInterfaceID)
	}
	return status, nil
}
