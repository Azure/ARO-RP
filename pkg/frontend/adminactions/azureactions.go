package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// AzureActions contains those actions which rely solely on Azure clients, not using any k8s clients
type AzureActions interface {
	ResourcesList(ctx context.Context) ([]byte, error)
	NICReconcileFailedState(ctx context.Context, nicName string) error
	VMRedeployAndWait(ctx context.Context, vmName string) error
	VMStartAndWait(ctx context.Context, vmName string) error
	VMStopAndWait(ctx context.Context, vmName string) error
	VMSizeList(ctx context.Context) ([]mgmtcompute.ResourceSku, error)
	VMResize(ctx context.Context, vmName string, vmSize string) error
	VMSerialConsole(ctx context.Context, w http.ResponseWriter, log *logrus.Entry, vmName string) error
}

type azureActions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	resources          features.ResourcesClient
	resourceSkus       compute.ResourceSkusClient
	virtualMachines    compute.VirtualMachinesClient
	virtualNetworks    network.VirtualNetworksClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	routeTables        network.RouteTablesClient
	storageAccounts    storage.AccountsClient
	networkInterfaces  network.InterfacesClient
}

// NewAzureActions returns an azureActions
func NewAzureActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument) (AzureActions, error) {
	fpAuth, err := env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID,
		env.Environment().ResourceManagerEndpoint)
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
		virtualNetworks:    network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		diskEncryptionSets: compute.NewDiskEncryptionSetsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		routeTables:        network.NewRouteTablesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		storageAccounts:    storage.NewAccountsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		networkInterfaces:  network.NewInterfacesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
	}, nil
}

func (a *azureActions) VMRedeployAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.RedeployAndWait(ctx, clusterRGName, vmName)
}

func (a *azureActions) VMStartAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.StartAndWait(ctx, clusterRGName, vmName)
}

func (a *azureActions) VMStopAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.StopAndWait(ctx, clusterRGName, vmName)
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
