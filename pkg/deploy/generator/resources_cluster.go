package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	diskEncryptionKeyName = "concat(resourceGroup().name, '-disk-encryption-key')"
	diskEncryptionSetName = "concat(resourceGroup().name, '-disk-encryption-set')"
)

func (g *generator) clusterVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "[parameters('vnetAddressPrefix')]", nil, "[parameters('ci')]", nil)
}

func (g *generator) clusterRouteTable() *arm.Resource {
	rt := &armnetwork.RouteTable{
		Properties: &armnetwork.RouteTablePropertiesFormat{
			Routes: []*armnetwork.Route{},
		},
		Name:     pointerutils.ToPtr("[concat(parameters('clusterName'), '-rt')]"),
		Type:     pointerutils.ToPtr("Microsoft.Network/routeTables"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   rt,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) clusterMasterSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.Subnet{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefixes: []*string{
					pointerutils.ToPtr("[parameters('masterAddressPrefix')]"),
				},
				RouteTable: &armnetwork.RouteTable{
					ID: pointerutils.ToPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: pointerutils.ToPtr("[concat('dev-vnet/', parameters('clusterName'), '-master')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
			"[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]",
		},
	}
}

func (g *generator) clusterWorkerSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.Subnet{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointerutils.ToPtr("[parameters('workerAddressPrefix')]"),
				RouteTable: &armnetwork.RouteTable{
					ID: pointerutils.ToPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: pointerutils.ToPtr("[concat('dev-vnet/', parameters('clusterName'), '-worker')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', concat(parameters('clusterName'), '-master'))]",
			"[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]",
		},
	}
}

func (g *generator) diskEncryptionKeyVault() *arm.Resource {
	vaultResource := g.keyVault("[parameters('kvName')]", &[]mgmtkeyvault.AccessPolicyEntry{}, "[parameters('ci')]", false, nil)

	return vaultResource
}

func (g *generator) diskEncryptionKey() *arm.Resource {
	key := &mgmtkeyvault.Key{
		KeyProperties: &mgmtkeyvault.KeyProperties{
			Kty:     mgmtkeyvault.RSA,
			KeySize: pointerutils.ToPtr(int32(4096)),
		},

		Name:     pointerutils.ToPtr(fmt.Sprintf("[concat(parameters('kvName'), '/', %s)]", diskEncryptionKeyName)),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults/keys"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   key,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{"[resourceId('Microsoft.KeyVault/vaults', parameters('kvName'))]"},
		Condition:  pointerutils.ToPtr("[parameters('ci')]"),
	}
}

func (g *generator) diskEncryptionSet() *arm.Resource {
	diskEncryptionSet := &mgmtcompute.DiskEncryptionSet{
		EncryptionSetProperties: &mgmtcompute.EncryptionSetProperties{
			ActiveKey: &mgmtcompute.KeyVaultAndKeyReference{
				KeyURL: pointerutils.ToPtr(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', parameters('kvName'), %s), '%s', 'Full').properties.keyUriWithVersion]", diskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: pointerutils.ToPtr("[resourceId('Microsoft.KeyVault/vaults', parameters('kvName'))]"),
				},
			},
		},

		Name:     pointerutils.ToPtr(fmt.Sprintf("[%s]", diskEncryptionSetName)),
		Type:     pointerutils.ToPtr("Microsoft.Compute/diskEncryptionSets"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
		Identity: &mgmtcompute.EncryptionSetIdentity{Type: mgmtcompute.SystemAssigned},
	}

	return &arm.Resource{
		Resource:   diskEncryptionSet,
		APIVersion: azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"),
		Condition:  pointerutils.ToPtr("[parameters('ci')]"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults/keys', parameters('kvName'), %s)]", diskEncryptionKeyName)},
	}
}

func (g *generator) diskEncryptionKeyVaultAccessPolicy() *arm.Resource {
	accessPolicy := &mgmtkeyvault.VaultAccessPolicyParameters{
		Properties: &mgmtkeyvault.VaultAccessPolicyProperties{
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: pointerutils.ToPtr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", diskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
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

		Name:     pointerutils.ToPtr("[concat(parameters('kvName'), '/add')]"),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		Condition:  pointerutils.ToPtr("[parameters('ci')]"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", diskEncryptionSetName)},
	}
}
