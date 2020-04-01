package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) removeBootstrap(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	i.log.Print("removing bootstrap vm")
	err := i.virtualmachines.DeleteAndWait(ctx, resourceGroup, "aro-bootstrap")
	if err != nil {
		return err
	}

	i.log.Print("removing bootstrap disk")
	err = i.disks.DeleteAndWait(ctx, resourceGroup, "aro-bootstrap_OSDisk")
	if err != nil {
		return err
	}

	i.log.Print("removing bootstrap nic")
	return i.interfaces.DeleteAndWait(ctx, resourceGroup, "aro-bootstrap-nic")
}

func (i *Installer) removeBootstrapIgnition(ctx context.Context) error {
	i.log.Print("remove ignition config")

	blobService, err := i.getBlobService(ctx, mgmtstorage.Permissions("d"), mgmtstorage.SignedResourceTypesC)
	if err != nil {
		return err
	}

	bootstrapIgn := blobService.GetContainerReference("ignition")
	_, err = bootstrapIgn.DeleteIfExists(&azstorage.DeleteContainerOptions{})
	return err
}
