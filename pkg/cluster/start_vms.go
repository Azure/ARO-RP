package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/virtualmachines"
)

// startVMs checks cluster VMs power state and starts deallocated and stopped VMs, if any
func (i *manager) startVMs(ctx context.Context) error {
	resourceGroupName := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vmsToStart, err := virtualmachines.ListStopped(ctx, i.virtualmachines, resourceGroupName)
	if err != nil {
		return err
	}

	g, groupCtx := errgroup.WithContext(ctx)
	for _, vm := range vmsToStart {
		vm := vm // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			return i.virtualmachines.StartAndWait(groupCtx, resourceGroupName, *vm.Name)
		})
	}
	return g.Wait()
}
