package dynamic

import (
	"context"
	"fmt"
	"net/http"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (dv *dynamic) ValidatePlatformWorkloadIdentityProfile(
	ctx context.Context,
	oc *api.OpenShiftCluster,
	platformWorkloadIdentityRolesByRoleName map[string]api.PlatformWorkloadIdentityRole,
	roleDefinitions armauthorization.RoleDefinitionsClient,
	clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient,
) error {
	dv.log.Print("ValidatePlatformWorkloadIdentityProfile")

	dv.platformIdentitiesActionsMap = map[string][]string{}
	dv.platformIdentities = map[string]api.PlatformWorkloadIdentity{}

	clusterResourceId, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return err
	}

	for k, pwi := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		role, ok := platformWorkloadIdentityRolesByRoleName[k]
		if ok {
			dv.platformIdentitiesActionsMap[k] = nil
			dv.platformIdentities[k] = pwi

			identityResourceId, err := azure.ParseResourceID(pwi.ResourceID)
			if err != nil {
				return err
			}

			expectedNames := map[string]struct{}{}

			for _, sa := range role.ServiceAccounts {
				expectedName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId, sa)
				expectedNames[expectedName] = struct{}{}
			}

			// validate federated identity credentials
			federatedCredentials, err := clusterMsiFederatedIdentityCredentials.List(ctx, identityResourceId.ResourceGroup, identityResourceId.ResourceName, &sdkmsi.FederatedIdentityCredentialsClientListOptions{})
			if err != nil {
				return err
			}

			for _, federatedCredential := range federatedCredentials {
				if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
						fmt.Sprintf("properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId", k),
						"Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure this identity is only used for this cluster and does not have any existing federated identity credentials.",
						*federatedCredential.Name,
						pwi.ResourceID,
						k,
					)
				}

				if _, ok := expectedNames[*federatedCredential.Name]; !ok {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
						fmt.Sprintf("properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId", k),
						"Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
						*federatedCredential.Name,
						pwi.ResourceID,
						k,
					)
				}
			}
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

// Validate that the cluster MSI has all permissions specified in AzureRedHatOpenShiftFederatedCredentialRole over each platform managed identity
func (dv *dynamic) ValidateClusterUserAssignedIdentity(ctx context.Context, platformIdentities map[string]api.PlatformWorkloadIdentity, roleDefinitions armauthorization.RoleDefinitionsClient) error {
	dv.log.Print("ValidateClusterUserAssignedIdentity")

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

		err = dv.validateActionsByOID(ctx, &pid, actions, nil)
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
