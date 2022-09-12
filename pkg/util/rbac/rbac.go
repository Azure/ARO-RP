package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	RoleACRPull                      = "7f951dda-4ed3-4680-a7ca-43fe172d538d"
	RoleContributor                  = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	RoleDocumentDBAccountContributor = "5bd9cd88-fe45-4216-938b-f97437e15450"
	RoleDNSZoneContributor           = "befefa01-2a29-4197-83a8-272ff33ce314"
	RoleNetworkContributor           = "4d97b98b-1d4f-4787-a291-c67834d212e7"
	RoleOwner                        = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	RoleReader                       = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
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
	r := &arm.Resource{
		Resource: mgmtauthorization.RoleAssignment{
			Name: to.StringPtr("[" + name + "]"),
			Type: to.StringPtr(resourceType + "/providers/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            to.StringPtr("[" + resourceID + "]"),
				RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '" + roleID + "')]"),
				PrincipalID:      to.StringPtr("[" + spID + "]"),
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
			Name: to.StringPtr("[" + name + "]"),
			Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            to.StringPtr("[resourceGroup().id]"),
				RoleDefinitionID: to.StringPtr("[" + resourceIDFunction + "('Microsoft.Authorization/roleDefinitions', " + roleID + ")]"),
				PrincipalID:      to.StringPtr("[" + spID + "]"),
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
