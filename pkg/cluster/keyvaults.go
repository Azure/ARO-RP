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

	// We want random characters from end of infraID to help ensure KV name uniqueness
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	suffixMaxLength := 20
	var suffix string

	// This runs after ensureInfraID bootstrap step, but just in case
	if infraID == "" {
		suffix = m.doc.OpenShiftCluster.Name
	} else {
		suffix = infraID
	}

	suffixLength := len(suffix)
	if suffixLength > suffixMaxLength {
		suffix = suffix[suffixLength-suffixMaxLength : suffixLength]
	}
	keyvaultName := "aro-" + suffix

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
