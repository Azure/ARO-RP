package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) NICReconcileFailedState(ctx context.Context, nicName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	nic, err := a.networkInterfaces.Get(ctx, clusterRGName, nicName, "")
	if err != nil {
		return err
	}

	// Ensure we only update NIC if in failed provisioning state
	if nic.ProvisioningState != mgmtnetwork.Failed {
		return fmt.Errorf("skipping nic '%s' because it is not in a failed provisioning state", nicName)
	}

	return a.networkInterfaces.CreateOrUpdateAndWait(ctx, clusterRGName, nicName, nic)
}
