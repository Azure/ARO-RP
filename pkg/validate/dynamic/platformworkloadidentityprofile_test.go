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
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	mock_remotepdp "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/authz/remotepdp"
	mock_armauthorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armauthorization"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	msiAllowedActions = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction1",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}

	msiNotAllowedActions = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: remotepdp.Denied,
			},
			{
				ActionId:       "FakeMSIDataAction1",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}

	msiActionMissing = remotepdp.AuthorizationDecisionResponse{
		Value: []remotepdp.AuthorizationDecision{
			{
				ActionId:       "FakeMSIAction1",
				AccessDecision: remotepdp.Allowed,
			},
			{
				ActionId:       "FakeMSIAction2",
				AccessDecision: remotepdp.Denied,
			},
			{
				ActionId:       "FakeMSIDataAction2",
				AccessDecision: remotepdp.Allowed,
			},
		},
	}

	msiRequiredPermissions = sdkauthorization.RoleDefinitionsClientGetByIDResponse{
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
	msiRequiredPermissionsList = []string{"FakeMSIAction1", "FakeMSIAction2", "FakeMSIDataAction1", "FakeMSIDataAction2"}

	platformIdentityRequiredPermissions = sdkauthorization.RoleDefinitionsClientGetByIDResponse{
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

	platformIdentityRequiredPermissionsList = []string{"FakeAction1", "FakeAction2", "FakeDataAction1", "FakeDataAction2"}
)

func TestValidatePlatformWorkloadIdentityProfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msiResourceID := resourceGroupID + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/miwi-msi-resource"
	dummyClientId := uuid.DefaultGenerator.Generate()
	dummyObjectId := uuid.DefaultGenerator.Generate()
	platformWorkloadIdentities := map[string]api.PlatformWorkloadIdentity{
		"Dummy2": {
			ResourceID: platformIdentity1,
		},
		"Dummy1": {
			ResourceID: platformIdentity1,
		},
	}
	desiredPlatformWorkloadIdentities := map[string]api.PlatformWorkloadIdentity{
		"Dummy1": {
			ResourceID: platformIdentity1,
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
			OperatorName: "Dummy1",
		},
	}
	openShiftVersion := "4.14.40"

	for _, tt := range []struct {
		name                             string
		platformIdentityRoles            map[string]api.PlatformWorkloadIdentityRole
		oc                               *api.OpenShiftCluster
		mocks                            func(*mock_armauthorization.MockRoleDefinitionsClient)
		wantPlatformIdentities           map[string]api.PlatformWorkloadIdentity
		wantPlatformIdentitiesActionsMap map[string][]string
		checkAccessMocks                 func(context.CancelFunc, *mock_remotepdp.MockRemotePDPClient, *mock_azcore.MockTokenCredential)
		wantErr                          string
	}{
		{
			name:                  "Success - Validation for the OC doc for PlatformWorkloadIdentityProfile",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy1": {
								ResourceID: platformIdentity1,
							},
						},
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Return(&msiAllowedActions, nil).AnyTimes()
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
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.15.40")),
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Return(&msiAllowedActions, nil).AnyTimes()
			},
			wantPlatformIdentities: map[string]api.PlatformWorkloadIdentity{
				"Dummy1": {
					ResourceID: platformIdentity1,
				},
			},
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name:                  "Success - Mismatch between desired and provided platform Identities - desired are fulfilled",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
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
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Return(&msiAllowedActions, nil).AnyTimes()
			},
			wantPlatformIdentities: desiredPlatformWorkloadIdentities,
			wantPlatformIdentitiesActionsMap: map[string][]string{
				"Dummy1": platformIdentityRequiredPermissionsList,
			},
		},
		{
			name: "Fail - UpgradeableTo is provided, but desired identities are not fulfilled",
			platformIdentityRoles: map[string]api.PlatformWorkloadIdentityRole{
				"Dummy3": {
					OperatorName: "Dummy3",
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.15.40")),
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
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
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.14.60")),
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
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
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
						UpgradeableTo:              ptr.To(api.UpgradeableTo("4.13.60")),
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy3]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - Mismatch between desired and provided platform Identities - count mismatch 2",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy1]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - Mismatch between desired and provided platform Identities - different operators",
			platformIdentityRoles: desiredPlatformWorkloadIdentitiesMap,
			oc: &api.OpenShiftCluster{
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
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			wantErr: fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[Dummy1]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name:                  "Fail - MSI Resource ID is invalid",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: platformWorkloadIdentities,
					},
					ClusterProfile: api.ClusterProfile{
						Version: openShiftVersion,
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: map[string]api.UserAssignedIdentity{
						"invalidUUID": {},
					},
				},
			},
			wantErr: "parsing failed for invalidUUID. Invalid resource Id format",
		},
		{
			name:                  "Fail - Getting role definition failed",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
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
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, errors.New("Generic Error"))
			},
			wantErr: "Generic Error",
		},
		{
			name:                  "Fail - Invalid Platform identity Resource ID",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"Dummy2": {
								ResourceID: "Invalid UUID",
							},
							"Dummy1": {
								ResourceID: "Invalid UUID",
							},
						},
					},
				},
				Identity: &api.ManagedServiceIdentity{
					UserAssignedIdentities: clusterMSI,
				},
			},
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
			},
			wantErr: "parsing failed for Invalid UUID. Invalid resource Id format",
		},
		{
			name:                  "Fail - An action is denied for a platform identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
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
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Do(func(arg0, arg1 interface{}) {
					cancel()
				}).Return(&msiNotAllowedActions, nil).AnyTimes()
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name:                  "Fail - An action is missing for a platform identity",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
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
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, nil)
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Do(func(arg0, arg1 interface{}) {
					cancel()
				}).Return(&msiActionMissing, nil).AnyTimes()
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name:                  "Fail - Getting Role Definition for Platform Identity Role returns error",
			platformIdentityRoles: validRolesForVersion,
			oc: &api.OpenShiftCluster{
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
			mocks: func(roleDefinitions *mock_armauthorization.MockRoleDefinitionsClient) {
				roleDefinitions.EXPECT().GetByID(ctx, rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole, &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).Return(msiRequiredPermissions, nil)
				roleDefinitions.EXPECT().GetByID(ctx, gomock.Any(), &sdkauthorization.RoleDefinitionsClientGetByIDOptions{}).AnyTimes().Return(platformIdentityRequiredPermissions, errors.New("Generic Error"))
			},
			checkAccessMocks: func(cancel context.CancelFunc, pdpClient *mock_remotepdp.MockRemotePDPClient, tokenCred *mock_azcore.MockTokenCredential) {
				mockTokenCredential(tokenCred)
				msiAuthReq := createAuthorizationRequest(dummyObjectId, platformIdentity1, msiRequiredPermissionsList...)
				pdpClient.EXPECT().CheckAccess(gomock.Any(), msiAuthReq).Return(&msiAllowedActions, errors.New("Generic Error")).AnyTimes()
			},
			wantErr: "Generic Error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			roleDefinitions := mock_armauthorization.NewMockRoleDefinitionsClient(controller)
			pdpClient := mock_remotepdp.NewMockRemotePDPClient(controller)

			dv := &dynamic{
				env:            _env,
				authorizerType: AuthorizerClusterServicePrincipal,
				log:            logrus.NewEntry(logrus.StandardLogger()),
				pdpClient:      pdpClient,
			}

			tokenCred := mock_azcore.NewMockTokenCredential(controller)

			if tt.checkAccessMocks != nil {
				tt.checkAccessMocks(cancel, pdpClient, tokenCred)
			}

			if tt.mocks != nil {
				tt.mocks(roleDefinitions)
			}

			err := dv.ValidatePlatformWorkloadIdentityProfile(ctx, tt.oc, tt.platformIdentityRoles, roleDefinitions)
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
