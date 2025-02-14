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
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	expectedAudience = "openshift"
)

func (dv *dynamic) ValidatePlatformWorkloadIdentityProfile(
	ctx context.Context,
	oc *api.OpenShiftCluster,
	platformWorkloadIdentityRolesByRoleName map[string]api.PlatformWorkloadIdentityRole,
	roleDefinitions armauthorization.RoleDefinitionsClient,
	clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient,
	platformWorkloadIdentities map[string]api.PlatformWorkloadIdentity, // Platform Workload Identities with object and client IDs
) (err error) {
	dv.log.Print("ValidatePlatformWorkloadIdentityProfile")

	dv.platformIdentitiesActionsMap = map[string][]string{}
	dv.platformIdentities = platformWorkloadIdentities

	// Check if any required platform identity is missing
	if len(dv.platformIdentities) != len(platformWorkloadIdentityRolesByRoleName) {
		return platformworkloadidentity.GetPlatformWorkloadIdentityMismatchError(oc, platformWorkloadIdentityRolesByRoleName)
	}

	for k, pwi := range dv.platformIdentities {
		role, exists := platformWorkloadIdentityRolesByRoleName[k]
		if !exists {
			return platformworkloadidentity.GetPlatformWorkloadIdentityMismatchError(oc, platformWorkloadIdentityRolesByRoleName)
		}

		roleDefinitionID := stringutils.LastTokenByte(role.RoleDefinitionID, '/')
		actions, err := getActionsForRoleDefinition(ctx, roleDefinitionID, roleDefinitions)
		if err != nil {
			return err
		}
		dv.platformIdentitiesActionsMap[role.OperatorName] = actions

		identityResourceId, err := azure.ParseResourceID(pwi.ResourceID)
		if err != nil {
			return err
		}

		// validate federated identity credentials
		federatedCredentials, err := clusterMsiFederatedIdentityCredentials.List(ctx, identityResourceId.ResourceGroup, identityResourceId.ResourceName, &sdkmsi.FederatedIdentityCredentialsClientListOptions{})
		if err != nil {
			return err
		}

		for _, federatedCredential := range federatedCredentials {
			switch {
			case federatedCredential == nil,
				federatedCredential.Name == nil,
				federatedCredential.Properties == nil:
				return fmt.Errorf("received invalid federated credential")
			case oc.Properties.ProvisioningState == api.ProvisioningStateCreating:
				return api.NewCloudError(
					http.StatusBadRequest,
					api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
					fmt.Sprintf("properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId", k),
					fmt.Sprintf(
						"Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure this identity is only used for this cluster and does not have any existing federated identity credentials.",
						*federatedCredential.Name,
						pwi.ResourceID,
						k,
					))
			case len(federatedCredential.Properties.Audiences) != 1,
				*federatedCredential.Properties.Audiences[0] != expectedAudience,
				federatedCredential.Properties.Issuer == nil,
				*federatedCredential.Properties.Issuer != string(*oc.Properties.ClusterProfile.OIDCIssuer):
				return api.NewCloudError(
					http.StatusBadRequest,
					api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
					fmt.Sprintf("properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId", k),
					fmt.Sprintf(
						"Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
						*federatedCredential.Name,
						pwi.ResourceID,
						k,
					))
			}
		}
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
