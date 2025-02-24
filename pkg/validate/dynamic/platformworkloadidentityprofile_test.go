package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/checkaccess-v2-go-sdk/client"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_armauthorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armauthorization"
	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_checkaccess "github.com/Azure/ARO-RP/pkg/util/mocks/checkaccess"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateClusterUserAssignedIdentity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	platformWorkloadIdentities := map[string]api.PlatformWorkloadIdentity{
		"Dummy1": {
			ResourceID: platformIdentity1,
		},
		"Dummy2": {
			ResourceID: platformIdentity1,
		},
	}

	msiAllowedActions := client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction1",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: client.Allowed,
			},
		},
	}

	msiNotAllowedActions := client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: client.Denied,
			},
			{
				ActionId:       "FakeMSIDataAction1",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: client.Allowed,
			},
		},
	}

	msiActionMissing := client.AuthorizationDecisionResponse{
		Value: []client.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: client.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: client.Denied,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: client.Allowed,
			},
		},
	}

	msiRequiredPermissions := sdkauthorization.RoleDefinitionsClientGetByIDResponse{
		RoleDefinition: sdkauthorization.RoleDefinition{
			Properties: &sdkauthorization.RoleDefinitionProperties{
				Permissions: []*sdkauthorization.Permission{
					{
						Actions: []*string{
							pointerutils.ToPtr("FakeMSIAction1"),
							pointerutils.ToPtr("FakeMSIAction2"),
						},
						DataActions: []*string{
							pointerutils.ToPtr("FakeMSIDataAction1"),
							pointerutils.ToPtr("FakeMSIDataAction2"),
						},
					},
				},
			},
		},
	}
	msiRequiredPermissionsList := []string{"FakeMSIAction1", "FakeMSIAction2", "FakeMSIDataAction1", "FakeMSIDataAction2"}
	msiAuthZRequest := client.AuthorizationRequest{
		Subject: client.SubjectInfo{
			Attributes: client.SubjectAttributes{
				ObjectId:  dummyObjectId,
				ClaimName: client.GroupExpansion,
			},
		},
		Actions:  []client.ActionInfo{{Id: "FakeMSIAction1"}, {Id: "FakeMSIAction2"}, {Id: "FakeMSIDataAction1"}, {Id: "FakeMSIDataAction2"}},
		Resource: client.ResourceInfo{Id: platformIdentity1},
	}

	for _, tt := range []struct {
		name               string
		platformIdentities map[string]api.PlatformWorkloadIdentity
		mocks              func(*mock_armauthorization.MockRoleDefinitionsClient)
		checkAccessMocks   func(context.CancelFunc, *mock_checkaccess.MockRemotePDPClient, *mock_azcore.MockTokenCredential, *mock_env.MockInterface)
		wantErr            string
	}{
		{
			name:               "Pass - All Cluster MSI has required permissions on platform workload identity",
			platformIdentities: platformWorkloadIdentities,
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					platformIdentity1,
					msiRequiredPermissionsList,
					validTestToken,
				).AnyTimes().Return(&msiAuthZRequest, nil)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthZRequest).Do(func(arg0, arg1 interface{}) {
					cancel()
				}).Return(&msiAllowedActions, nil).AnyTimes()
			},
		},
		{
			name:               "Fail - Get permission definition failed with generic error",
			platformIdentities: platformWorkloadIdentities,
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, errors.New("Generic Error"))
			},
			wantErr: "Generic Error",
		},
		{
			name: "Fail - Invalid Platform identity Resource ID",
			platformIdentities: map[string]api.PlatformWorkloadIdentity{
				"Dummy2": {
					ResourceID: "Invalid UUID",
				},
				"Dummy1": {
					ResourceID: "Invalid UUID",
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
			},
			wantErr: "parsing failed for Invalid UUID. Invalid resource Id format",
		},
		{
			name:               "Fail - An action is denied for a platform identity",
			platformIdentities: platformWorkloadIdentities,
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					platformIdentity1,
					msiRequiredPermissionsList,
					validTestToken,
				).AnyTimes().Return(&msiAuthZRequest, nil)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthZRequest).Do(func(arg0, arg1 interface{}) {
					cancel()
				}).Return(&msiNotAllowedActions, nil).AnyTimes()
			},
			wantErr: "context canceled",
		},
		{
			name:               "Fail - An action is missing for a platform identity",
			platformIdentities: platformWorkloadIdentities,
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_checkaccess.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential, env *mock_env.MockInterface) {
				mockTokenCredential(tokenCred)
				env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
				pdpClient.EXPECT().CreateAuthorizationRequest(
					platformIdentity1,
					msiRequiredPermissionsList,
					validTestToken,
				).AnyTimes().Return(&msiAuthZRequest, nil)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthZRequest).Do(func(arg0, arg1 interface{}) {
					cancel()
				}).Return(&msiActionMissing, nil).AnyTimes()
			},
			wantErr: "context canceled",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			roleDefinitions := mock_armauthorization.NewMockRoleDefinitionsClient(controller)
			pdpClient := mock_checkaccess.NewMockRemotePDPClient(controller)
			tokenCred := mock_azcore.NewMockTokenCredential(controller)

			dv := &dynamic{
				env:                        _env,
				authorizerType:             AuthorizerClusterUserAssignedIdentity,
				log:                        logrus.NewEntry(logrus.StandardLogger()),
				pdpClient:                  pdpClient,
				checkAccessSubjectInfoCred: tokenCred,
			}

			if tt.checkAccessMocks != nil {
				tt.checkAccessMocks(cancel, pdpClient, tokenCred, _env)
			}

			if tt.mocks != nil {
				tt.mocks(roleDefinitions)
			}

			err := dv.ValidateClusterUserAssignedIdentity(ctx, tt.platformIdentities, roleDefinitions)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidatePlatformWorkloadIdentityProfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msiResourceID := resourceGroupID + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/miwi-msi-resource"
	clusterResourceId, _ := azure.ParseResourceID(clusterID)
	platformIdentity1ResourceId, _ := azure.ParseResourceID(platformIdentity1)
	expectedPlatformIdentity1FederatedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, platformIdentity1ResourceId, platformIdentity1SAName)
	expectedOIDCIssuer := "https://fakeissuer.fakedomain/fakecluster"
	platformWorkloadIdentities := map[string]api.PlatformWorkloadIdentity{
		"Dummy1": {
			ResourceID: platformIdentity1,
		},
	}
	desiredPlatformWorkloadIdentities := map[string]api.PlatformWorkloadIdentity{
		"Dummy1": {
			ResourceID: platformIdentity1,
			ObjectID:   dummyObjectId,
			ClientID:   dummyClientId,
		},
	}
	desiredPlatformWorkloadIdentitiesMap := map[string]api.PlatformWorkloadIdentityRole{
		"Dummy1": {
			OperatorName: "Dummy1",
		},
	}
	clusterMSI := map[string]api.UserAssignedIdentity{
		msiResourceID: {
			ClientID:    dummyClientId,
			PrincipalID: dummyObjectId,
		},
	}
	validRolesForVersion := map[string]api.PlatformWorkloadIdentityRole{
		"Dummy1": {
			OperatorName:    "Dummy1",
			ServiceAccounts: []string{platformIdentity1SAName},
		},
	}
	openShiftVersion := "4.14.40"
	platformIdentityRequiredPermissions := sdkauthorization.RoleDefinitionsClientGetByIDResponse{
		RoleDefinition: sdkauthorization.RoleDefinition{
			Properties: &sdkauthorization.RoleDefinitionProperties{
				Permissions: []*sdkauthorization.Permission{
					{
						Actions: []*string{
							pointerutils.ToPtr("FakeAction1"),
							pointerutils.ToPtr("FakeAction2"),
						},
						DataActions: []*string{
							pointerutils.ToPtr("FakeDataAction1"),
							pointerutils.ToPtr("FakeDataAction2"),
						},
					},
				},
			},
		},
	}

	platformIdentityRequiredPermissionsList := []string{"FakeAction1", "FakeAction2", "FakeDataAction1", "FakeDataAction2"}

	for _, tt := range []struct {
		name                             string
		platformIdentityRoles            map[string]api.PlatformWorkloadIdentityRole
		oc                               *api.OpenShiftCluster
		mocks                            func(*mock_armauthorization.MockRoleDefinitionsClient, *mock_armmsi.MockFederatedIdentityCredentialsClient)
		wantPlatformIdentities           map[string]api.PlatformWorkloadIdentity
		wantPlatformIdentitiesActionsMap map[string][]string
		wantErr                          string
	}{
		{
			name:                  "Success - Validation for the OC doc for PlatformWorkloadIdentityProfile",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Success - Existing Federated Identity Credentials are for this cluster",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)

				expectedPlatformIdentity1FederatedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, platformIdentity1ResourceId, platformIdentity1SAName)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &expectedPlatformIdentity1FederatedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Success - Existing Federated Identity Credentials are for this cluster but for an unknown service account",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)

				expectedPlatformIdentity1FederatedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, platformIdentity1ResourceId, platformIdentity1SAName)
				expectedPlatformIdentity1ExtraFederatedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, platformIdentity1ResourceId, "system:serviceaccount:something:else")

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &expectedPlatformIdentity1FederatedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
						{
							Name: &expectedPlatformIdentity1ExtraFederatedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   pointerutils.ToPtr("something else"),
							},
						},
					}, nil)
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Success - UpgradeableTo is provided",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.15.40")),
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Success - Mismatch between desired and provided platform Identities - desired are fulfilled",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Fail - Error when listing federated identity credentials",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return(nil, fmt.Errorf("something unexpected occurred"))
			},
			wantErr: "something unexpected occurred",
		},
		{
			name:                  "Fail - Unexpected Federated Identity Credential (wrong audience) found on platform workload identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: pointerutils.ToPtr("something-else"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("something else")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: fmt.Sprintf(
				"400: %s: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId: Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
				api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
				"Dummy1",
				"something-else",
				platformIdentity1,
				"Dummy1",
			),
		},
		{
			name:                  "Fail - Unexpected Federated Identity Credential (missing audience) found on platform workload identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: pointerutils.ToPtr("something-else"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: nil,
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: fmt.Sprintf(
				"400: %s: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId: Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
				api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
				"Dummy1",
				"something-else",
				platformIdentity1,
				"Dummy1",
			),
		},
		{
			name:                  "Fail - Unexpected Federated Identity Credential (wrong issuer) found on platform workload identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: pointerutils.ToPtr("something-else"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    pointerutils.ToPtr("something else"),
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: fmt.Sprintf(
				"400: %s: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId: Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
				api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
				"Dummy1",
				"something-else",
				platformIdentity1,
				"Dummy1",
			),
		},
		{
			name:                  "Fail - Unexpected Federated Identity Credential (missing issuer) found on platform workload identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: pointerutils.ToPtr("something-else"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    nil,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: fmt.Sprintf(
				"400: %s: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId: Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure only federated credentials provisioned by the ARO service for this cluster are present.",
				api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
				"Dummy1",
				"something-else",
				platformIdentity1,
				"Dummy1",
			),
		},
		{
			name:                  "Fail - A Federated Identity Credential found on platform workload identity during creation",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState: api.ProvisioningStateCreating,
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &expectedPlatformIdentity1FederatedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: fmt.Sprintf(
				"400: %s: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities.%s.resourceId: Unexpected federated credential '%s' found on platform workload identity '%s' used for role '%s'. Please ensure this identity is only used for this cluster and does not have any existing federated identity credentials.",
				api.CloudErrorCodePlatformWorkloadIdentityContainsInvalidFederatedCredential,
				"Dummy1",
				expectedPlatformIdentity1FederatedCredName,
				platformIdentity1,
				"Dummy1",
			),
		},
		{
			name:                  "Fail - Federated Identity Credential client returns nil credential",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{nil}, nil)
			},
			wantErr: "received invalid federated credential",
		},
		{
			name:                  "Fail - Federated Identity Credential client returns credential with nil name",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: nil,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &expectedOIDCIssuer,
								Subject:   &platformIdentity1SAName,
							},
						},
					}, nil)
			},
			wantErr: "received invalid federated credential",
		},
		{
			name:                  "Fail - Federated Identity Credential client returns credential with nil properties",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(platformIdentity1ResourceId.ResourceGroup), gomock.Eq(platformIdentity1ResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{{Name: &expectedPlatformIdentity1FederatedCredName}}, nil)
			},
			wantErr: "received invalid federated credential",
		},
		{
			name: "Fail - UpgradeableTo is provided, but desired identities are not fulfilled",
			platformIdentityRoles: map[string]api.PlatformWorkloadIdentityRole{
				"Dummy3": {
					OperatorName: "Dummy3",
				},
			},
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.15.40")),
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s or %s'. The required platform workload identities are '[Dummy3]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14", "4.15"),
		},
		{
			name: "Fail - UpgradeableTo is provided(ignored because minor version is equal to cluster minor version), but desired identities are not fulfilled",
			platformIdentityRoles: map[string]api.PlatformWorkloadIdentityRole{
				"Dummy3": {
					OperatorName: "Dummy3",
				},
			},
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.14.60")),
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy3]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name: "Fail - UpgradeableTo is provided(ignored because upgradeable version is smaller than cluster version), but desired identities are not fulfilled",
			platformIdentityRoles: map[string]api.PlatformWorkloadIdentityRole{
				"Dummy3": {
					OperatorName: "Dummy3",
				},
			},
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.13.60")),
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy3]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - Mismatch between desired and provided platform Identities - count mismatch and operator missing",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy1]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - Mismatch between desired and provided platform Identities - different operators",
			platformIdentityRoles: desiredPlatformWorkloadIdentitiesMap,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy2": {
								ResourceID: platformIdentity1,
							},
							"Dummy3": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy1]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - Getting role definition failed",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, errors.New("Generic Error"))
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			wantErr: "Generic Error",
		},
		{
			name:                  "Fail - Invalid Platform identity Resource ID",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: "Invalid UUID",
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			wantErr: "parsing failed for Invalid UUID. Invalid resource Id format",
		},
		{
			name:                  "Fail - Getting Role Definition for Platform Identity Role returns error",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
					},
					ClusterProfile: api.ClusterProfile{
						Version:    openShiftVersion,
						OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer(expectedOIDCIssuer)),
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient, federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, errors.New("Generic Error"))
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			wantErr: "Generic Error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			roleDefinitions := mock_armauthorization.NewMockRoleDefinitionsClient(controller)
			federatedIdentityCredentials := mock_armmsi.NewMockFederatedIdentityCredentialsClient(controller)

			dv := &dynamic{
				env:            _env,
				authorizerType: AuthorizerWorkloadIdentity,
				log:            logrus.NewEntry(logrus.StandardLogger()),
			}

			if tt.mocks != nil {
				tt.mocks(roleDefinitions, federatedIdentityCredentials)
			}

			pwis := tt.oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities
			updatedIdentities := make(map[string]api.PlatformWorkloadIdentity, len(pwis))

			for operatorName, pwi := range pwis {
				updatedIdentities[operatorName] = api.PlatformWorkloadIdentity{
					ResourceID: pwi.ResourceID,
					ClientID:   dummyClientId,
					ObjectID:   dummyObjectId,
				}
			}

			err := dv.ValidatePlatformWorkloadIdentityProfile(ctx, tt.oc, tt.platformIdentityRoles, roleDefinitions, federatedIdentityCredentials, updatedIdentities)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantPlatformIdentities != nil && !reflect.DeepEqual(tt.wantPlatformIdentities, dv.platformIdentities) {
				t.Fatalf("Expected platform identities are not populated in dv object")
			}
			if tt.wantPlatformIdentitiesActionsMap != nil && !reflect.DeepEqual(tt.wantPlatformIdentitiesActionsMap, dv.platformIdentitiesActionsMap) {
				t.Fatalf("Expected platform identities to permissions mapping is not populated in dv object")
			}
		})
	}
}
