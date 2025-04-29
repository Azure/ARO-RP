package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) NICReconcileFailedState(ctx context.Context, nicName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	nic, err := a.armNetworkInterfaces.Get(ctx, clusterRGName, nicName, nil)
	if err != nil {
		return err
	}

	// Ensure we only update NIC if in failed provisioning state
	if *nic.Properties.ProvisioningState != armnetwork.ProvisioningStateFailed {
		return fmt.Errorf("skipping nic '%s' because it is not in a failed provisioning state", nicName)
	}

	return a.armNetworkInterfaces.CreateOrUpdateAndWait(ctx, clusterRGName, nicName, nic.Interface, nil)
}
