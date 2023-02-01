package cluster

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (m *manager) createKeyvault(ctx context.Context) error {
	keyvaultName := "aro-secrets-" + m.doc.OpenShiftCluster.Name[:12]
	vaultNameAvailabilityParameters := keyvault.VaultCheckNameAvailabilityParameters{
		Name: &keyvaultName,
		Type: to.StringPtr("Microsoft.KeyVault/vaults"),
	}

	result, err := m.keyvaults.CheckNameAvailability(ctx, vaultNameAvailabilityParameters)
	if err != nil {
		return err
	}
	if result.NameAvailable != nil && !*result.NameAvailable {
		return fmt.Errorf("could not generate unique key vault name: %v", result.Reason)
	}

	keyvaultProperties := new(keyvault.VaultProperties)

	keyvaultParameters := keyvault.VaultCreateOrUpdateParameters{
		Location:   &m.doc.OpenShiftCluster.Location,
		Tags:       make(map[string]*string),
		Properties: keyvaultProperties,
	}
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	_, err = m.keyvaults.CreateOrUpdate(ctx, resourceGroup, keyvaultName, keyvaultParameters)
	if err != nil {
		return err
	}

	return nil
}
