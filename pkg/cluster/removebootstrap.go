package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) removeBootstrap(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	m.log.Print("removing bootstrap vm")
	err := m.virtualMachines.DeleteAndWait(ctx, resourceGroup, infraID+"-bootstrap", nil)
	if err != nil {
		return err
	}

	m.log.Print("removing bootstrap disk")
	err = m.disks.DeleteAndWait(ctx, resourceGroup, infraID+"-bootstrap_OSDisk")
	if err != nil {
		return err
	}

	m.log.Print("removing bootstrap nic")
	return m.interfaces.DeleteAndWait(ctx, resourceGroup, infraID+"-bootstrap-nic")
}

func (m *manager) removeBootstrapIgnition(ctx context.Context) error {
	m.log.Print("remove ignition config")

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, mgmtstorage.Permissions("d"), mgmtstorage.SignedResourceTypesC)
	if err != nil {
		return err
	}

	bootstrapIgn := blobService.GetContainerReference("ignition")
	_, err = bootstrapIgn.DeleteIfExists(&azstorage.DeleteContainerOptions{})
	return err
}
