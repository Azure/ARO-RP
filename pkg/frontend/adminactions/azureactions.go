package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
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
	VMSizeList(ctx context.Context) ([]mgmtcompute.ResourceSku, error)
	VMResize(ctx context.Context, vmName string, vmSize string) error
	ResourceGroupHasVM(ctx context.Context, vmName string) (bool, error)
	VMSerialConsole(ctx context.Context, log *logrus.Entry, vmName string, target io.Writer) error
	ResourceDeleteAndWait(ctx context.Context, resourceID string) error
}

type azureActions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	resources          features.ResourcesClient
	resourceSkus       compute.ResourceSkusClient
	virtualMachines    compute.VirtualMachinesClient
	virtualNetworks    armnetwork.VirtualNetworksClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	routeTables        armnetwork.RouteTablesClient
	storageAccounts    storage.AccountsClient
	networkInterfaces  network.InterfacesClient
	loadBalancers      armnetwork.LoadBalancersClient
	securityGroups     armnetwork.SecurityGroupsClient
}

// NewAzureActions returns an azureActions
func NewAzureActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument) (AzureActions, error) {
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

	routeTables, err := armnetwork.NewRouteTablesClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	virtualNetworks, err := armnetwork.NewVirtualNetworksClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	securityGroups, err := armnetwork.NewSecurityGroupsClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	loadBalancers, err := armnetwork.NewLoadBalancersClient(subscriptionDoc.ID, credential, options)
	if err != nil {
		return nil, err
	}

	return &azureActions{
		log: log,
		env: env,
		oc:  oc,

		resources:          features.NewResourcesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		resourceSkus:       compute.NewResourceSkusClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualMachines:    compute.NewVirtualMachinesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualNetworks:    virtualNetworks,
		diskEncryptionSets: compute.NewDiskEncryptionSetsClientWithAROEnvironment(env.Environment(), subscriptionDoc.ID, fpAuth),
		routeTables:        routeTables,
		storageAccounts:    storage.NewAccountsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		networkInterfaces:  network.NewInterfacesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		loadBalancers:      loadBalancers,
		securityGroups:     securityGroups,
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

func (a *azureActions) VMSizeList(ctx context.Context) ([]mgmtcompute.ResourceSku, error) {
	filter := fmt.Sprintf("location eq '%s'", a.env.Location())
	return a.resourceSkus.List(ctx, filter)
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
