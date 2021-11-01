package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (g *generator) diskEncryptionKeyVault() *arm.Resource {
	vaultResource := g.keyVault("[parameters('kvName')]", &[]mgmtkeyvault.AccessPolicyEntry{
		{
			ObjectID: to.StringPtr("[parameters('clusterServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Keys: &[]mgmtkeyvault.KeyPermissions{mgmtkeyvault.KeyPermissionsCreate, mgmtkeyvault.KeyPermissionsGet, mgmtkeyvault.KeyPermissionsDelete},
			},
			// is later replaced by "[subscription().tenantId]"
			TenantID: &tenantUUIDHack,
		},
	})

	vaultResource.Condition = to.StringPtr("[parameters('ci')]")

	return vaultResource
}

const diskEncryptionKeyName string = "concat(resourceGroup().name, '-', 'disk-encryption-key')"

func (g *generator) diskEncryptionKey() *arm.Resource {
	key := &mgmtkeyvault.Key{
		KeyProperties: &mgmtkeyvault.KeyProperties{
			Kty:     mgmtkeyvault.RSA,
			KeySize: to.Int32Ptr(4096),
		},

		Name:     to.StringPtr(fmt.Sprintf("[concat(parameters('kvName'), '/', %s)]", diskEncryptionKeyName)),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults/keys"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   key,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{"[resourceId('Microsoft.KeyVault/vaults', parameters('kvName'))]"},
		Condition:  to.StringPtr("[parameters('ci')]"),
	}
}

func (g *generator) diskEncryptionSet() *arm.Resource {
	diskEncryptionSet := &mgmtcompute.DiskEncryptionSet{
		EncryptionSetProperties: &mgmtcompute.EncryptionSetProperties{
			ActiveKey: &mgmtcompute.KeyVaultAndKeyReference{
				KeyURL: to.StringPtr(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', parameters('kvName'), %s), '%s', 'Full').properties.keyUriWithVersion]", diskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: to.StringPtr("[resourceId('Microsoft.KeyVault/vaults', parameters('kvName'))]"),
				},
			},
		},

		Name:     to.StringPtr("[parameters('diskEncryptionSetName')]"),
		Type:     to.StringPtr("Microsoft.Compute/diskEncryptionSets"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Identity: &mgmtcompute.EncryptionSetIdentity{Type: mgmtcompute.SystemAssigned},
	}

	return &arm.Resource{
		Resource:   diskEncryptionSet,
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		Condition:  to.StringPtr("[parameters('ci')]"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults/keys', parameters('kvName'), %s)]", diskEncryptionKeyName)},
	}
}

func (g *generator) diskEncryptionKeyVaultAccessPolicy() *arm.Resource {
	accessPolicy := &mgmtkeyvault.VaultAccessPolicyParameters{
		Properties: &mgmtkeyvault.VaultAccessPolicyProperties{
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: to.StringPtr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', parameters('diskEncryptionSetName')), '%s', 'Full').identity.PrincipalId]", azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
					Permissions: &mgmtkeyvault.Permissions{
						Keys: &[]mgmtkeyvault.KeyPermissions{
							mgmtkeyvault.KeyPermissionsGet,
							mgmtkeyvault.KeyPermissionsWrapKey,
							mgmtkeyvault.KeyPermissionsUnwrapKey,
						},
					},
				},
			},
		},

		Name:     to.StringPtr("[concat(parameters('kvName'), '/add')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{"[resourceId('Microsoft.Compute/diskEncryptionSets', parameters('diskEncryptionSetName'))]"},
	}
}
