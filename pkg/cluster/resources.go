package cluster

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/go-autorest/autorest/to"
)

var extraDenyAssignmentExclusions = map[string][]string{
	"Microsoft.RedHatOpenShift/RedHatEngineering": {
		"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
	},
}

func (m *manager) denyAssignments(clusterSPObjectID string) *arm.Resource {
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
						ID:   &clusterSPObjectID,
						Type: to.StringPtr("ServicePrincipal"),
					},
				},
				IsSystemProtected: to.BoolPtr(true),
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization/denyAssignments"),
	}
}

func (m *manager) clusterServicePrincipalRBAC(clusterSPObjectID string) *arm.Resource {
	return rbac.ResourceGroupRoleAssignmentWithName(
		rbac.RoleContributor,
		"'"+clusterSPObjectID+"'",
		"guid(resourceGroup().id, 'SP / Contributor')",
	)
}
