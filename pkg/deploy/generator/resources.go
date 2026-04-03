package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func (g *generator) actionGroup(name string, shortName string) *arm.Resource {
	return &arm.Resource{
		Resource: mgmtinsights.ActionGroupResource{
			ActionGroup: &mgmtinsights.ActionGroup{
				Enabled:        new(true),
				GroupShortName: &shortName,
			},
			Name:     &name,
			Type:     new("Microsoft.Insights/actionGroups"),
			Location: new("Global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights"),
	}
}

func (g *generator) dnsZone(name string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtdns.Zone{
			ZoneProperties: &mgmtdns.ZoneProperties{},
			Name:           &name,
			Type:           new("Microsoft.Network/dnsZones"),
			Location:       new("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/dnsZones"),
	}
}

func (g *generator) securityGroup(name string, securityRules []*armnetwork.SecurityRule, condition any) *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.SecurityGroup{
			Properties: &armnetwork.SecurityGroupPropertiesFormat{
				SecurityRules: securityRules,
			},
			Name:     &name,
			Type:     new("Microsoft.Network/networkSecurityGroups"),
			Location: new("[resourceGroup().location]"),
		},
		Condition:  condition,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) securityRules(name string, properties *armnetwork.SecurityRulePropertiesFormat, condition any) *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.SecurityRule{
			Properties: properties,
			Name:       &name,
			Type:       new("Microsoft.Network/networkSecurityGroups/securityRules"),
		},
		Location:   "[resourceGroup().location]",
		Condition:  condition,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) publicIPAddress(name string) *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.PublicIPAddress{
			SKU: &armnetwork.PublicIPAddressSKU{
				Name: pointerutils.ToPtr(armnetwork.PublicIPAddressSKUNameStandard),
			},
			Properties: &armnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodStatic),
			},
			Zones:    []*string{},
			Name:     &name,
			Type:     new("Microsoft.Network/publicIPAddresses"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) storageAccount(name string, accountProperties *mgmtstorage.AccountProperties, tags map[string]*string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.Account{
			Name:     &name,
			Type:     new("Microsoft.Storage/storageAccounts"),
			Kind:     mgmtstorage.KindStorageV2,
			Location: new("[resourceGroup().location]"),
			Sku: &mgmtstorage.Sku{
				Name: "Standard_LRS",
			},
			AccountProperties: accountProperties,
			Tags:              tags,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (g *generator) storageAccountBlobContainer(name string, storageAccountName string, containerProperties *mgmtstorage.ContainerProperties) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.BlobContainer{
			Name:                new("[" + name + "]"),
			Type:                new("Microsoft.Storage/storageAccounts/blobServices/containers"),
			ContainerProperties: containerProperties,
		},
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Storage/storageAccounts', %s)]", storageAccountName)},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (g *generator) virtualNetwork(name, addressPrefix string, subnets []*armnetwork.Subnet, condition any, dependsOn []string) *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.VirtualNetwork{
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{
						new(addressPrefix),
					},
				},
				Subnets: subnets,
			},
			Name:     &name,
			Type:     new("Microsoft.Network/virtualNetworks"),
			Location: new("[resourceGroup().location]"),
		},
		Condition:  condition,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn:  dependsOn,
	}
}

// virtualNetworkPeering configures vnetA to peer with vnetB, two symmetrical
// configurations have to be applied for a peering to work
func (g *generator) virtualNetworkPeering(name, vnetB string, allowGatewayTransit, useRemoteGateways bool, dependsOn []string) *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.VirtualNetworkPeering{
			Properties: &armnetwork.VirtualNetworkPeeringPropertiesFormat{
				AllowVirtualNetworkAccess: new(true),
				AllowForwardedTraffic:     new(true),
				AllowGatewayTransit:       new(allowGatewayTransit),
				UseRemoteGateways:         new(useRemoteGateways),
				RemoteVirtualNetwork: &armnetwork.SubResource{
					ID: &vnetB,
				},
			},
			Name: &name,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		Type:       "Microsoft.Network/virtualNetworks/virtualNetworkPeerings",
		Location:   "[resourceGroup().location]",
		DependsOn:  dependsOn,
	}
}

func (g *generator) keyVault(name string, accessPolicies *[]mgmtkeyvault.AccessPolicyEntry, condition any, enableEntraIdRbac bool, dependsOn []string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtkeyvault.Vault{
			Properties: &mgmtkeyvault.VaultProperties{
				EnableRbacAuthorization:  new(enableEntraIdRbac),
				EnablePurgeProtection:    new(true),
				EnabledForDiskEncryption: new(true),
				Sku: &mgmtkeyvault.Sku{
					Name:   mgmtkeyvault.Standard,
					Family: new("A"),
				},
				// is later replaced by "[subscription().tenantId]"
				TenantID:       &tenantUUIDHack,
				AccessPolicies: accessPolicies,
			},
			Name:     &name,
			Type:     new("Microsoft.KeyVault/vaults"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		Condition:  condition,
		DependsOn:  dependsOn,
	}
}
