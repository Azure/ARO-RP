package dynamic

import (
	"context"
	"fmt"
	"net/http"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (dv *dynamic) ValidatePlatformWorkloadIdentityProfile(ctx context.Context, oc *api.OpenShiftCluster, platformWorkloadIdentityRolesByRoleName map[string]api.PlatformWorkloadIdentityRole, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	dv.log.Print("ValidatePlatformWorkloadIdentityProfile")

	dv.platformIdentitiesActionsMap = map[string][]string{}
	dv.platformIdentities = map[string]api.PlatformWorkloadIdentity{}

	for k, pwi := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		_, ok := platformWorkloadIdentityRolesByRoleName[k]
		if ok {
			dv.platformIdentitiesActionsMap[k] = nil
			dv.platformIdentities[k] = pwi
		}
	}

	// Check if any required platform identity is missing
	if len(dv.platformIdentities) != len(platformWorkloadIdentityRolesByRoleName) {
		requiredOperatorIdentities := []string{}
		for _, role := range platformWorkloadIdentityRolesByRoleName {
			requiredOperatorIdentities = append(requiredOperatorIdentities, role.OperatorName)
		}
		currentOpenShiftVersion, err := version.ParseVersion(oc.Properties.ClusterProfile.Version)
		if err != nil {
			return err
		}
		currentMinorVersion := currentOpenShiftVersion.MinorVersion()
		v := currentMinorVersion
		if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
			upgradeableVersion, err := version.ParseVersion(string(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo))
			if err != nil {
				return err
			}
			upgradeableMinorVersion := upgradeableVersion.MinorVersion()
			if currentMinorVersion != upgradeableMinorVersion && currentOpenShiftVersion.Lt(upgradeableVersion) {
				v = fmt.Sprintf("%s or %s", v, upgradeableMinorVersion)
			}
		}
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePlatformWorkloadIdentityMismatch,
			"properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities", "There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '%v'", v, requiredOperatorIdentities)
	}

	err := dv.validateClusterMSI(ctx, oc, roleDefinitions)
	if err != nil {
		return err
	}

	for _, role := range platformWorkloadIdentityRolesByRoleName {
		roleDefinitionID := stringutils.LastTokenByte(role.RoleDefinitionID, '/')
		actions, err := getActionsForRoleDefinition(ctx, roleDefinitionID, roleDefinitions)
		if err != nil {
			return err
		}

		dv.platformIdentitiesActionsMap[role.OperatorName] = actions
	}

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

// Validate that the cluster MSI has all permissions specified in AzureRedHatOpenShiftFederatedCredentialRole over each platform managed identity
func (dv *dynamic) validateClusterMSIPermissions(ctx context.Context, oid string, platformIdentities map[string]api.PlatformWorkloadIdentity, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	actions, err := getActionsForRoleDefinition(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, roleDefinitions)
	if err != nil {
		return err
	}

	for name, platformIdentity := range platformIdentities {
		dv.log.Printf("validateClusterMSIPermissions for %s", name)
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
