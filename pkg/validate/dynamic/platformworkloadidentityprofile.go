package dynamic

import (
	"context"
	"net/http"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (dv *dynamic) ValidatePlatformWorkloadIdentityProfile(ctx context.Context, oc *api.OpenShiftCluster, platformWorkloadIdentityRoles []api.PlatformWorkloadIdentityRole, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	dv.log.Print("ValidatePlatformWorkloadIdentityProfile")

	platformIdentitiesActionsMap := map[string][]string{}

	for _, role := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		platformIdentitiesActionsMap[role.OperatorName] = nil
	}

	roleIsMissing := false
	requiredOperatorIdentities := []string{}
	for _, role := range platformWorkloadIdentityRoles {
		requiredOperatorIdentities = append(requiredOperatorIdentities, role.OperatorName)
		_, ok := platformIdentitiesActionsMap[role.OperatorName]
		if !ok {
			roleIsMissing = true
		}
	}

	if roleIsMissing || len(platformWorkloadIdentityRoles) != len(platformIdentitiesActionsMap) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePlatformWorkloadIdentityMismatch,
			"properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities", "There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift version '%s'. The required platform workload identities are '%v'", oc.Properties.ClusterProfile.Version, requiredOperatorIdentities)
	}

	err := dv.validateClusterMSI(ctx, oc, roleDefinitions)
	if err != nil {
		return err
	}

	for _, role := range platformWorkloadIdentityRoles {
		actions, err := getActionsForRoleDefinition(ctx, role.RoleDefinitionID, roleDefinitions)
		if err != nil {
			return err
		}

		platformIdentitiesActionsMap[role.OperatorName] = actions
	}

	dv.platformIdentities = oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities
	dv.platformIdentitiesActionsMap = platformIdentitiesActionsMap

	return nil
}

func (dv *dynamic) validateClusterMSI(ctx context.Context, oc *api.OpenShiftCluster, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	for resourceID, identity := range oc.Identity.UserAssignedIdentities {
		_, err := azure.ParseResourceID(resourceID)
		if err != nil {
			return err
		}

		return dv.validateClusterMSIPermissions(ctx, identity.PrincipalID, oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities, roleDefinitions)
	}

	return nil
}

func (dv *dynamic) validateClusterMSIPermissions(ctx context.Context, oid string, platformIdentities []api.PlatformWorkloadIdentity, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	actions, err := getActionsForRoleDefinition(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, roleDefinitions)
	if err != nil {
		return err
	}

	for _, platformIdentity := range platformIdentities {
		dv.log.Printf("validateClusterMSIPermissions for %s", platformIdentity.OperatorName)
		pid, err := azure.ParseResourceID(platformIdentity.ResourceID)
		if err != nil {
			return err
		}

		err = dv.validateActionsByOID(ctx, &pid, actions, &oid)
		if err != nil {
			return err
		}
	}
	return nil
}

func getActionsForRoleDefinition(ctx context.Context, roleDefinitionID string, roleDefinitions armauthorization.RoleDefinitionsClient) ([]string, error) {
	definition, err := roleDefinitions.GetByID(ctx, roleDefinitionID, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{})
	if err != nil {
		return nil, err
	}

	if len(definition.RoleDefinition.Properties.Permissions) == 0 {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError,
			"", "Internal server error.")
	}

	actions := []string{}
	for _, action := range definition.RoleDefinition.Properties.Permissions[0].Actions {
		actions = append(actions, *action)
	}

	for _, dataAction := range definition.RoleDefinition.Properties.Permissions[0].DataActions {
		actions = append(actions, *dataAction)
	}
	return actions, nil
}
