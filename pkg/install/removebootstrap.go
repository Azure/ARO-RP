package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
)

func (i *Installer) removeBootstrap(ctx context.Context) error {
	resourceGroup := i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')+1:]
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
	err = i.interfaces.DeleteAndWait(ctx, resourceGroup, "aro-bootstrap-nic")
	if err != nil {
		return err
	}
	return nil
}
