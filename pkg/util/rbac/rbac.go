package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	RoleACRPull                                     = "7f951dda-4ed3-4680-a7ca-43fe172d538d"
	RoleContributor                                 = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	RoleDocumentDBAccountContributor                = "5bd9cd88-fe45-4216-938b-f97437e15450"
	RoleDocumentDBDataContributor                   = "00000000-0000-0000-0000-000000000002"
	RoleDNSZoneContributor                          = "befefa01-2a29-4197-83a8-272ff33ce314"
	RoleNetworkContributor                          = "4d97b98b-1d4f-4787-a291-c67834d212e7"
	RoleOwner                                       = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	RoleReader                                      = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
	RoleStorageAccountContributor                   = "17d1049b-9a84-46fb-8f53-869881c3d3ab"
	RoleStorageBlobDataContributor                  = "ba92f5b4-2d11-453d-a403-e96b0029c9fe"
	RoleKeyVaultSecretsOfficer                      = "b86a8fe4-44ce-4948-aee5-eccb2c155cd7"
	RoleAzureRedHatOpenShiftFederatedCredentialRole = "ef318e2a-8334-4a05-9e4a-295a196c6a6e"
)

// ResourceRoleAssignment returns a Resource granting roleID on the resource of
// type resourceType with name resourceName to spID.  Arguments resourceName and
// spID must be valid ARM expressions, e.g. "'foo'" or "concat('foo')".  Use
// this function in new ARM templates.
func ResourceRoleAssignment(roleID, spID, resourceType, resourceName string, condition ...interface{}) *arm.Resource {
	resourceID := "resourceId('" + resourceType + "', " + resourceName + ")"

	return ResourceRoleAssignmentWithName(roleID, spID, resourceType, resourceName, "concat("+resourceName+", '/Microsoft.Authorization/', guid("+resourceID+", "+spID+", '"+roleID+"'))", condition...)
}

// ResourceRoleAssignmentWithName returns a Resource granting roleID on the
// resource of type resourceType with name resourceName to spID.  Arguments
// resourceName, spID and name must be valid ARM expressions, e.g. "'foo'" or
// "concat('foo')".  Use this function in ARM templates which have already been
// deployed, to preserve the name and avoid a RoleAssignmentExists error.
func ResourceRoleAssignmentWithName(roleID, spID, resourceType, resourceName, name string, condition ...interface{}) *arm.Resource {
	resourceID := "resourceId('" + resourceType + "', " + resourceName + ")"

	var roleDefinitionID string
	if strings.HasPrefix(roleID, "parameters") {
		roleDefinitionID = "[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', " + roleID + ")]"
	} else {
		roleDefinitionID = "[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '" + roleID + "')]"
	}
	r := &arm.Resource{
		Resource: mgmtauthorization.RoleAssignment{
			Name: pointerutils.ToPtr("[" + name + "]"),
			Type: pointerutils.ToPtr(resourceType + "/providers/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            pointerutils.ToPtr("[" + resourceID + "]"),
				RoleDefinitionID: pointerutils.ToPtr(roleDefinitionID),
				PrincipalID:      pointerutils.ToPtr("[" + spID + "]"),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization"),
		DependsOn: []string{
			"[" + resourceID + "]",
		},
	}

	if len(condition) > 0 {
		r.Condition = condition[0]
	}

	return r
}

func ResourceRoleAssignmentWithScope(roleID, spID, resourceType string, scope string, names string, condition ...interface{}) *arm.Resource {
	r := &arm.Resource{
		Resource: mgmtauthorization.RoleAssignment{
			Name: pointerutils.ToPtr("[" + names + "]"),
			Type: pointerutils.ToPtr(resourceType + "/providers/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            pointerutils.ToPtr("[" + scope + "]"),
				RoleDefinitionID: pointerutils.ToPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '" + roleID + "')]"),
				PrincipalID:      pointerutils.ToPtr("[" + spID + "]"),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization"),
		DependsOn: []string{
			"[" + scope + "]",
		},
	}

	if len(condition) > 0 {
		r.Condition = condition[0]
	}

	return r
}

// ResourceGroupRoleAssignment returns a Resource granting roleID on the current
// resource group to spID.  Argument spID must be a valid ARM expression, e.g.
// "'foo'" or "concat('foo')".  Use this function in new ARM templates.
func ResourceGroupRoleAssignment(roleID, spID string, condition ...interface{}) *arm.Resource {
	return ResourceGroupRoleAssignmentWithName(roleID, spID, "guid(resourceGroup().id, '"+spID+"', '"+roleID+"')", condition...)
}

func resourceGroupRoleAssignmentWithDetails(roleID, spID string, name string, dependsOn []string, subscriptionScope bool, condition ...interface{}) *arm.Resource {
	resourceIDFunction := "resourceId"
	if subscriptionScope {
		resourceIDFunction = "subscriptionResourceId"
	}
	r := &arm.Resource{
		Resource: mgmtauthorization.RoleAssignment{
			Name: pointerutils.ToPtr("[" + name + "]"),
			Type: pointerutils.ToPtr("Microsoft.Authorization/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            pointerutils.ToPtr("[resourceGroup().id]"),
				RoleDefinitionID: pointerutils.ToPtr("[" + resourceIDFunction + "('Microsoft.Authorization/roleDefinitions', " + roleID + ")]"),
				PrincipalID:      pointerutils.ToPtr("[" + spID + "]"),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization"),
		DependsOn:  dependsOn,
	}

	if len(condition) > 0 {
		r.Condition = condition[0]
	}

	return r
}

// ResourceGroupRoleAssignmentWithName returns a Resource granting roleID on the
// current resource group to spID.  Arguments spID and name must be valid ARM
// expressions, e.g. "'foo'" or "concat('foo')".  Use this function in ARM
// templates which have already been deployed, to preserve the name and avoid a
// RoleAssignmentExists error.
func ResourceGroupRoleAssignmentWithName(roleID, spID string, name string, condition ...interface{}) *arm.Resource {
	return resourceGroupRoleAssignmentWithDetails("'"+roleID+"'", spID, name, nil, true, condition...)
}
