package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

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
	VMRedeployAndWait(ctx context.Context, vmName string) error
	VMSerialConsole(ctx context.Context, w http.ResponseWriter, log *logrus.Entry, vmName string) error
}

type azureActions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	resources         features.ResourcesClient
	virtualMachines   compute.VirtualMachinesClient
	virtualNetworks   network.VirtualNetworksClient
	diskEncryptionSet compute.DiskEncryptionSetsClient
	routeTables       network.RouteTablesClient
	storageAccounts   storage.AccountsClient
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

		resources:         features.NewResourcesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualMachines:   compute.NewVirtualMachinesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		virtualNetworks:   network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		diskEncryptionSet: compute.NewDiskEncryptionSetsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		routeTables:       network.NewRouteTablesClient(env.Environment(), subscriptionDoc.ID, fpAuth),
		storageAccounts:   storage.NewAccountsClient(env.Environment(), subscriptionDoc.ID, fpAuth),
	}, nil
}

func (a *azureActions) VMRedeployAndWait(ctx context.Context, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	return a.virtualMachines.RedeployAndWait(ctx, clusterRGName, vmName)
}
