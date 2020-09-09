package infra

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"math/rand"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	routeTableName   = "routetable"
	vnetName         = "dev-vnet"
	masterSubnetName = "master-subnet"
	workerSubnetName = "worker-subbet"
)

func networkVnet() *arm.Resource {
	randPrefix := func() string {
		return fmt.Sprintf("10.%d.%d.0/24", rand.Intn(127), rand.Intn(127))
	}

	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.0.0/9",
					},
				},
				Subnets: &[]mgmtnetwork.Subnet{
					{
						Name: to.StringPtr(masterSubnetName),
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr(randPrefix()),
							ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
								{
									Service: to.StringPtr("Microsoft.ContainerRegistry"),
								},
							},
							PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
							RouteTable: &mgmtnetwork.RouteTable{
								Name: to.StringPtr(routeTableName),
								ID:   to.StringPtr(fmt.Sprintf("[resourceid('Microsoft.Network/routeTables', '%s')]", routeTableName)),
							},
						},
					},
					{
						Name: to.StringPtr(workerSubnetName),
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr(randPrefix()),
							ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
								{
									Service: to.StringPtr("Microsoft.ContainerRegistry"),
								},
							},
						},
					},
				},
			},
			Name:     to.StringPtr(vnetName),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			fmt.Sprintf("[resourceid('Microsoft.Network/routeTables', '%s')]", routeTableName),
		},
	}
}

func networkRouteTable() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.RouteTable{
			Name:                       to.StringPtr(routeTableName),
			Type:                       to.StringPtr("Microsoft.Network/routeTables"),
			Location:                   to.StringPtr("[resourceGroup().location]"),
			RouteTablePropertiesFormat: &mgmtnetwork.RouteTablePropertiesFormat{},
		},
		APIVersion: "2020-05-01",
	}
}

func rbacVnetRoleAssignment(spID, roleAssignmentName string) *arm.Resource {
	vnetResourceID := fmt.Sprintf("[resourceid('Microsoft.Network/virtualNetworks', '%s')]", vnetName)

	return &arm.Resource{
		Resource: &mgmtauthorization.RoleAssignment{
			Name: to.StringPtr(fmt.Sprintf("[concat('%[1]s', '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/virtualNetworks', '%[1]s'), '%[2]s'))]", vnetName, roleAssignmentName)),
			Type: to.StringPtr("Microsoft.Network/virtualNetworks/providers/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            to.StringPtr(vnetResourceID),
				RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"),
				PrincipalID:      to.StringPtr(spID),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		DependsOn: []string{
			vnetResourceID,
		},
	}
}

func rbacRouteTableRoleAssignment(spID, roleAssignmentName string) *arm.Resource {
	rtResourceID := fmt.Sprintf("[resourceid('Microsoft.Network/routeTables', '%s')]", routeTableName)

	return &arm.Resource{
		Resource: mgmtauthorization.RoleAssignment{
			Name: to.StringPtr(fmt.Sprintf("[concat('%[1]s', '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/routeTables', '%[1]s'), '%[2]s'))]", routeTableName, roleAssignmentName)),
			Type: to.StringPtr("Microsoft.Network/routeTables/providers/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            to.StringPtr(rtResourceID),
				RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"),
				PrincipalID:      to.StringPtr(spID),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		DependsOn: []string{
			rtResourceID,
		},
	}
}

func rbacResourceGroupRoleAssignment(spID string, resourceGroupName string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtauthorization.RoleAssignment{
			Name: to.StringPtr(fmt.Sprintf("[guid(resourceGroup().id, '%s')]", resourceGroupName)),
			Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            to.StringPtr("[resourcegroup().id]"),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
				PrincipalID:      to.StringPtr(spID),
				RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"),
			},
		},
		APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
	}
}
