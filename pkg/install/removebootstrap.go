package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

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
