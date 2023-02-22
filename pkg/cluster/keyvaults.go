package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
)

func (m *manager) createKeyvault(ctx context.Context) error {
	// We want random characters from end of infraID to help ensure KV name uniqueness
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	suffixMaxLength := 20
        suffix := infraID
        
	// This runs after ensureInfraID bootstrap step, but just in case
	if infraID == "" {
		suffix = m.doc.OpenShiftCluster.Name
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

	tenantID, err := uuid.FromString(m.env.TenantID())
	if err != nil {
		return err
	}

	accessPolicies := []keyvault.AccessPolicyEntry{}

	keyvaultProperties := keyvault.VaultProperties{
		TenantID: &tenantID,
		Sku: &keyvault.Sku{
			Name:   keyvault.Standard,
			Family: to.StringPtr("A"),
		},
		AccessPolicies: &accessPolicies,
	}

	keyvaultParameters := keyvault.VaultCreateOrUpdateParameters{
		Location:   &m.doc.OpenShiftCluster.Location,
		Tags:       make(map[string]*string),
		Properties: &keyvaultProperties,
	}
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	_, err = m.keyvaults.CreateOrUpdate(ctx, resourceGroup, keyvaultName, keyvaultParameters)
	if err != nil {
		return err
	}

	return nil
}
