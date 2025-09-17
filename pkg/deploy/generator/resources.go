package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v7"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
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
				Enabled:        pointerutils.ToPtr(true),
				GroupShortName: &shortName,
			},
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.Insights/actionGroups"),
			Location: pointerutils.ToPtr("Global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights"),
	}
}

func (g *generator) dnsZone(name string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtdns.Zone{
			ZoneProperties: &mgmtdns.ZoneProperties{},
			Name:           &name,
			Type:           pointerutils.ToPtr("Microsoft.Network/dnsZones"),
			Location:       pointerutils.ToPtr("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/dnsZones"),
	}
}

func (g *generator) networkSecurityPerimeter(name string) *arm.Resource {

	return &arm.Resource{
		Resource: &sdknetwork.SecurityPerimeter{
			Location:   pointerutils.ToPtr("[resourceGroup().location]"),
			Properties: &sdknetwork.SecurityPerimeterProperties{},
			Name:       &name,
			Type:       pointerutils.ToPtr("Microsoft.Network/networkSecurityPerimeters"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/networkSecurityPerimeters"),
	}
}

// networkSecurityPerimeterProfile creates a new nsp profile with the hardcoded name `default`.
func (g *generator) networkSecurityPerimeterProfile(nspName string) *arm.Resource {
	return &arm.Resource{
		Resource: &sdknetwork.NspProfile{
			Name: pointerutils.ToPtr(fmt.Sprintf("%s/default", nspName)),
			Type: pointerutils.ToPtr("Microsoft.Network/networkSecurityPerimeters/profiles"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/networkSecurityPerimeters/profiles"),
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.Network/networkSecurityPerimeters', '%s')]", nspName),
		},
	}
}

func (g *generator) networkSecurityPerimeterAssociation(nspName string, associationName string, targetResourceId string) *arm.Resource {
	return &arm.Resource{
		Resource: &sdknetwork.NspAssociation{
			Properties: &sdknetwork.NspAssociationProperties{
				AccessMode: pointerutils.ToPtr(sdknetwork.AssociationAccessModeLearning),
				PrivateLinkResource: &sdknetwork.SubResource{
					ID: &targetResourceId,
				},
				Profile: &sdknetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.Network/networkSecurityPerimeters/profiles', '%s/default')]", nspName)),
				},
			},
			Name: pointerutils.ToPtr(fmt.Sprintf("%s/%s", nspName, associationName)),
			Type: pointerutils.ToPtr("Microsoft.Network/networkSecurityPerimeters/resourceAssociations"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/networkSecurityPerimeters/resourceAssociations"),
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.Network/networkSecurityPerimeters/profiles', '%s/default')]", nspName),
		},
	}
}

func (g *generator) securityGroup(name string, securityRules *[]mgmtnetwork.SecurityRule, condition interface{}) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.SecurityGroup{
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
				SecurityRules: securityRules,
			},
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.Network/networkSecurityGroups"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		Condition:  condition,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) securityRules(name string, properties *mgmtnetwork.SecurityRulePropertiesFormat, condition interface{}) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.SecurityRule{
			SecurityRulePropertiesFormat: properties,
			Name:                         &name,
			Type:                         pointerutils.ToPtr("Microsoft.Network/networkSecurityGroups/securityRules"),
		},
		Location:   "[resourceGroup().location]",
		Condition:  condition,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) publicIPAddress(name string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: mgmtnetwork.Static,
			},
			Zones:    &[]string{},
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.Network/publicIPAddresses"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) storageAccount(name string, accountProperties *mgmtstorage.AccountProperties, tags map[string]*string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.Account{
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.Storage/storageAccounts"),
			Kind:     mgmtstorage.KindStorageV2,
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
			Name:                pointerutils.ToPtr("[" + name + "]"),
			Type:                pointerutils.ToPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
			ContainerProperties: containerProperties,
		},
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Storage/storageAccounts', %s)]", storageAccountName)},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (g *generator) virtualNetwork(name, addressPrefix string, subnets *[]mgmtnetwork.Subnet, condition interface{}, dependsOn []string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						addressPrefix,
					},
				},
				Subnets: subnets,
			},
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.Network/virtualNetworks"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
		Resource: &mgmtnetwork.VirtualNetworkPeering{
			VirtualNetworkPeeringPropertiesFormat: &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
				AllowVirtualNetworkAccess: pointerutils.ToPtr(true),
				AllowForwardedTraffic:     pointerutils.ToPtr(true),
				AllowGatewayTransit:       pointerutils.ToPtr(allowGatewayTransit),
				UseRemoteGateways:         pointerutils.ToPtr(useRemoteGateways),
				RemoteVirtualNetwork: &mgmtnetwork.SubResource{
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

func (g *generator) keyVault(name string, accessPolicies *[]mgmtkeyvault.AccessPolicyEntry, condition interface{}, enableEntraIdRbac bool, dependsOn []string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtkeyvault.Vault{
			Properties: &mgmtkeyvault.VaultProperties{
				EnableRbacAuthorization:  pointerutils.ToPtr(enableEntraIdRbac),
				EnablePurgeProtection:    pointerutils.ToPtr(true),
				EnabledForDiskEncryption: pointerutils.ToPtr(true),
				Sku: &mgmtkeyvault.Sku{
					Name:   mgmtkeyvault.Standard,
					Family: pointerutils.ToPtr("A"),
				},
				// is later replaced by "[subscription().tenantId]"
				TenantID:       &tenantUUIDHack,
				AccessPolicies: accessPolicies,
			},
			Name:     &name,
			Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		Condition:  condition,
		DependsOn:  dependsOn,
	}
}
