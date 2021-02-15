package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

var extraDenyAssignmentExclusions = map[string][]string{
	"Microsoft.RedHatOpenShift/RedHatEngineering": {
		"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
	},
}

func (m *manager) denyAssignments() *arm.Resource {
	notActions := []string{
		"Microsoft.Network/networkSecurityGroups/join/action",
		"Microsoft.Compute/disks/beginGetAccess/action",
		"Microsoft.Compute/disks/endGetAccess/action",
		"Microsoft.Compute/disks/write",
		"Microsoft.Compute/snapshots/beginGetAccess/action",
		"Microsoft.Compute/snapshots/endGetAccess/action",
		"Microsoft.Compute/snapshots/write",
		"Microsoft.Compute/snapshots/delete",
	}

	var props = m.subscriptionDoc.Subscription.Properties

	for flag, exclusions := range extraDenyAssignmentExclusions {
		if feature.IsRegisteredForFeature(props, flag) {
			notActions = append(notActions, exclusions...)
		}
	}

	return &arm.Resource{
		Resource: &mgmtauthorization.DenyAssignment{
			Name: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
			Type: to.StringPtr("Microsoft.Authorization/denyAssignments"),
			DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
				DenyAssignmentName: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
				Permissions: &[]mgmtauthorization.DenyAssignmentPermission{
					{
						Actions: &[]string{
							"*/action",
							"*/delete",
							"*/write",
						},
						NotActions: &notActions,
					},
				},
				Scope: &m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
				Principals: &[]mgmtauthorization.Principal{
					{
						ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
						Type: to.StringPtr("SystemDefined"),
					},
				},
				ExcludePrincipals: &[]mgmtauthorization.Principal{
					{
						ID:   &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID,
						Type: to.StringPtr("ServicePrincipal"),
					},
				},
				IsSystemProtected: to.BoolPtr(true),
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization/denyAssignments"),
	}
}

func (m *manager) clusterServicePrincipalRBAC() *arm.Resource {
	return rbac.ResourceGroupRoleAssignmentWithName(
		rbac.RoleContributor,
		"'"+m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID+"'",
		"guid(resourceGroup().id, 'SP / Contributor')",
	)
}

func (m *manager) clusterStorageAccount(region string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.Account{
			Sku: &mgmtstorage.Sku{
				Name: "Standard_LRS",
			},
			Name:     to.StringPtr("cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix),
			Location: &region,
			Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (m *manager) clusterStorageAccountBlob(name string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.BlobContainer{
			Name: to.StringPtr("cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/" + name),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
		DependsOn: []string{
			"Microsoft.Storage/storageAccounts/cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix,
		},
	}
}
