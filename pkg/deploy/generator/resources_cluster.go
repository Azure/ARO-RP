package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	diskEncryptionKeyName = "concat(resourceGroup().name, '-disk-encryption-key')"
	diskEncryptionSetName = "concat(resourceGroup().name, '-disk-encryption-set')"
)

func (g *generator) clusterVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "[parameters('vnetAddressPrefix')]", nil, "[parameters('ci')]", nil)
}

func (g *generator) clusterRouteTable() *arm.Resource {
	rt := &mgmtnetwork.RouteTable{
		RouteTablePropertiesFormat: &mgmtnetwork.RouteTablePropertiesFormat{
			Routes: &[]mgmtnetwork.Route{},
		},
		Name:     to.StringPtr("[concat(parameters('clusterName'), '-rt')]"),
		Type:     to.StringPtr("Microsoft.Network/routeTables"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   rt,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) clusterMasterSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('masterAddressPrefix')]"),
				RouteTable: &mgmtnetwork.RouteTable{
					ID: to.StringPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-master')]"),
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
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('workerAddressPrefix')]"),
				RouteTable: &mgmtnetwork.RouteTable{
					ID: to.StringPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-worker')]"),
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
	vaultResource := g.keyVault("[parameters('kvName')]", &[]mgmtkeyvault.AccessPolicyEntry{}, "[parameters('ci')]", nil)

	return vaultResource
}

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

		Name:     to.StringPtr(fmt.Sprintf("[%s]", diskEncryptionSetName)),
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
					ObjectID: to.StringPtr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", diskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
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
		Condition:  to.StringPtr("[parameters('ci')]"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", diskEncryptionSetName)},
	}
}
