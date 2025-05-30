package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	azsecretssdk "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	gofrsuuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_azuresdk "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_msi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/msi"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	instanceID                 = "testID"
	rgName                     = "testRG"
	location                   = "testLocation"
	globalRGName               = "testRG-global"
	subscriptionRGName         = "testRG-subscription"
	notExistingFileName        = "testFile"
	existingFileName           = generator.FileGatewayProductionPredeploy
	existingFileDeploymentName = strings.TrimSuffix(existingFileName, ".json")
	secretExists               = "secretExists"
	noSecretExists             = "noSecretExists"
	vmssName                   = rpVMSSPrefix + "test"
	invalidVMSSName            = "other-vmss"

	errGeneric            = errors.New("generic error")
	deploymentFailedError = &azure.ServiceError{
		Code:    "DeploymentFailed",
		Details: []map[string]interface{}{{}},
	}
	deploymentNotFoundError = autorest.DetailedError{
		Original: &azure.RequestError{
			ServiceError: &azure.ServiceError{
				Code:    "DeploymentNotFound",
				Details: []map[string]interface{}{{}},
			},
		},
	}

	healthyVMSS = mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{Code: to.StringPtr("HealthState/healthy")},
		},
	}
	unhealthyVMSS = mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/unhealthy"),
			},
		},
	}

	newSecretBundle = azsecretssdk.GetSecretResponse{
		Secret: azsecretssdk.Secret{
			Attributes: &azsecretssdk.SecretAttributes{Created: pointerutils.ToPtr(time.Now())},
		},
	}

	listForItems = func(items []*azsecretssdk.SecretProperties) []azsecretssdk.ListSecretPropertiesResponse {
		return []azsecretssdk.ListSecretPropertiesResponse{{
			SecretPropertiesListResult: azsecretssdk.SecretPropertiesListResult{
				Value: items,
			},
		}}
	}

	secretItems = func(err error) *runtime.Pager[azsecretssdk.ListSecretPropertiesResponse] {
		return mock_azuresdk.NewPager(listForItems([]*azsecretssdk.SecretProperties{{ID: pointerutils.ToPtr(azsecretssdk.ID("https://myvaultname.vault.azure.net/keys/secretExists/whatever"))}}), []error{err})
	}

	vmsss        = []mgmtcompute.VirtualMachineScaleSet{{Name: to.StringPtr(vmssName)}}
	invalidVMSSs = []mgmtcompute.VirtualMachineScaleSet{{Name: &invalidVMSSName}}
	vms          = []mgmtcompute.VirtualMachineScaleSetVM{{InstanceID: to.StringPtr(instanceID)}}
)

func TestPreDeploy(t *testing.T) {
	ctx := context.Background()
	rpRgName := "testRG-aro-rp"
	gatewayRgName := "testRG-gwy"
	overrideLocation := "overrideTestLocation"
	globalDevopsManagedIdentity := "aro-test-devops"

	group := mgmtfeatures.ResourceGroup{
		Location: &location,
	}
	fakeMSIObjectId, _ := gofrsuuid.NewV4()
	msi := mgmtmsi.Identity{
		UserAssignedIdentityProperties: &mgmtmsi.UserAssignedIdentityProperties{PrincipalID: &fakeMSIObjectId},
	}
	deployment := mgmtfeatures.DeploymentExtended{}
	vmsss := []mgmtcompute.VirtualMachineScaleSet{{Name: &vmssName}}
	oneMissingSecrets := []string{env.FrontendEncryptionSecretV2Name, env.PortalServerSessionKeySecretName, env.EncryptionSecretName, env.FrontendEncryptionSecretName, env.PortalServerSSHKeySecretName}
	oneMissingSecretItems := []*azsecretssdk.SecretProperties{}
	for _, secret := range oneMissingSecrets {
		oneMissingSecretItems = append(oneMissingSecretItems, &azsecretssdk.SecretProperties{ID: pointerutils.ToPtr(azsecretssdk.ID("https://myvaultname.vault.azure.net/keys/" + secret + "/whatever"))})
	}

	type resourceGroups struct {
		subscriptionRGName       string
		globalResourceGroup      string
		rpResourceGroupName      string
		gatewayResourceGroupName string
	}
	type testParams struct {
		resourceGroups              resourceGroups
		location                    string
		instanceID                  string
		vmssName                    string
		restartScript               string
		overrideLocation            string
		globalDevopsManagedIdentity string
		acrReplicaDisabled          bool
	}
	type mock func(*mock_features.MockDeploymentsClient, *mock_features.MockResourceGroupsClient, *mock_msi.MockUserAssignedIdentitiesClient, *mock_azsecrets.MockClient, *mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	createOrUpdateAtSubscriptionScopeAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, "rp-global-subscription-"+tp.location, gomock.Any()).Return(returnError)
		}
	}
	createOrUpdateAndWaitMock := func(resourceGroup string, returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, resourceGroup, gomock.Any(), gomock.Any()).Return(returnError)
		}
	}
	createOrUpdateMock := func(resourceGroup string, returnResourceGroup mgmtfeatures.ResourceGroup, returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			rg.EXPECT().CreateOrUpdate(ctx, resourceGroup, mgmtfeatures.ResourceGroup{Location: &tp.location}).Return(returnResourceGroup, returnError)
		}
	}
	msiGetMock := func(resourceGroup string, returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			m.EXPECT().Get(ctx, resourceGroup, gomock.Any()).Return(msi, returnError)
		}
	}
	getDeploymentMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			d.EXPECT().Get(ctx, tp.resourceGroups.gatewayResourceGroupName, gomock.Any()).Return(deployment, returnError)
		}
	}
	getSecretsMock := func(items []*azsecretssdk.SecretProperties, returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			k.EXPECT().NewListSecretPropertiesPager(nil).Return(mock_azuresdk.NewPager(listForItems(items), []error{returnError}))
		}
	}
	getSecretMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		k.EXPECT().GetSecret(ctx, gomock.Any(), "", nil).Return(newSecretBundle, nil)
	}
	setSecretMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, nil)
	}
	vmssListMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmss.EXPECT().List(ctx, gomock.Any()).Return(vmsss, nil)
	}
	vmssVMsListMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().List(ctx, gomock.Any(), tp.vmssName, "", "", "").Return(vms, nil)
	}
	vmRestartMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, gomock.Any(), tp.vmssName, tp.instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{tp.restartScript},
		}).Return(nil)
	}
	instanceViewMock := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), gomock.Any(), tp.vmssName, tp.instanceID).Return(healthyVMSS, nil)
	}

	for _, tt := range []struct {
		name               string
		acrReplicaDisabled bool
		testParams         testParams
		mocks              []mock
		wantErr            string
	}{
		{
			name: "don't continue if Global Subscription RBAC DeploymentFailed",
			testParams: testParams{
				location:                    location,
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if Global Subscription RBAC Deployment is Successful but SubscriptionResourceGroup creation fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName: subscriptionRGName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if SubscriptionResourceGroup creation is Successful but GlobalResourceGroup creation fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:  subscriptionRGName,
					globalResourceGroup: globalRGName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if GlobalResourceGroup creation is Successful but RPResourceGroup creation fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:  subscriptionRGName,
					globalResourceGroup: globalRGName,
					rpResourceGroupName: rpRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if RPResourceGroup creation is successful but GatewayResourceGroup creation fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if GatewayResourceGroup is successful but rp-subscription template deployment fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if rp-subscription template deployment is successful but rp managed identity get fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if rp managed identity get is successful but gateway managed identity get fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if rp and gateway managed identity gets are successful but devops managed identity get fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if rpglobal deployment fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if rpglobal deployment fails twice with DeploymentFailed",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, deploymentFailedError), createOrUpdateAndWaitMock(globalRGName, deploymentFailedError),
			},
			wantErr: `Code="DeploymentFailed" Message="" Details=[{}]`,
		},
		{
			name: "don't continue if ACR Replication fails",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            overrideLocation,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if skipping ACR Replication due to no ACRLocationOverride but failing gateway predeploy",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if skipping ACR Replication due to ACRLocationOverride same as GlobalResourceGroupLocation but failing gateway predeploy",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            location,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue if skipping ACR Replication due to ACRReplicaDisabled but failing gateway predeploy",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            overrideLocation,
				acrReplicaDisabled:          true,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "don't continue gateway predeploy is successful but rp predeploy failed",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            overrideLocation,
				acrReplicaDisabled:          true,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, nil), createOrUpdateAndWaitMock(rpRgName, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "get error for the configureServiceSecrets",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            overrideLocation,
				acrReplicaDisabled:          true,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, nil), createOrUpdateAndWaitMock(rpRgName, nil), getSecretsMock(oneMissingSecretItems, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "Everything is successful",
			testParams: testParams{
				location: location,
				resourceGroups: resourceGroups{
					subscriptionRGName:       subscriptionRGName,
					globalResourceGroup:      globalRGName,
					rpResourceGroupName:      rpRgName,
					gatewayResourceGroupName: gatewayRgName,
				},
				globalDevopsManagedIdentity: globalDevopsManagedIdentity,
				overrideLocation:            overrideLocation,
				acrReplicaDisabled:          true,
				vmssName:                    vmssName,
				instanceID:                  instanceID,
				restartScript:               rpRestartScript,
			},
			mocks: []mock{
				createOrUpdateAtSubscriptionScopeAndWaitMock(nil), createOrUpdateMock(subscriptionRGName, group, nil), createOrUpdateMock(globalRGName, group, nil), createOrUpdateMock(rpRgName, group, nil), createOrUpdateMock(gatewayRgName, group, nil), createOrUpdateAndWaitMock(subscriptionRGName, nil), createOrUpdateAndWaitMock(rpRgName, nil), msiGetMock(rpRgName, nil), createOrUpdateAndWaitMock(gatewayRgName, nil), msiGetMock(gatewayRgName, nil), msiGetMock(globalRGName, nil), createOrUpdateAndWaitMock(globalRGName, nil), getDeploymentMock(deploymentNotFoundError), createOrUpdateAndWaitMock(gatewayRgName, nil), createOrUpdateAndWaitMock(rpRgName, nil), getSecretsMock(oneMissingSecretItems, nil), setSecretMock, getSecretsMock(oneMissingSecretItems, nil), getSecretMock, getSecretsMock(oneMissingSecretItems, nil), getSecretMock, getSecretsMock(oneMissingSecretItems, nil), getSecretsMock(oneMissingSecretItems, nil), getSecretsMock(oneMissingSecretItems, nil), vmssListMock, vmssVMsListMock, vmRestartMock, instanceViewMock,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)
			mockResourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			mockMSIs := mock_msi.NewMockUserAssignedIdentitiesClient(controller)
			mockKV := mock_azsecrets.NewMockClient(controller)
			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			mockVMSSVM := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:                          logrus.NewEntry(logrus.StandardLogger()),
				globaldeployments:            mockDeployments,
				deployments:                  mockDeployments,
				groups:                       mockResourceGroups,
				globalgroups:                 mockResourceGroups,
				userassignedidentities:       mockMSIs,
				globaluserassignedidentities: mockMSIs,
				config: &RPConfig{
					Configuration: &Configuration{
						GlobalResourceGroupLocation:       &tt.testParams.location,
						SubscriptionResourceGroupLocation: &tt.testParams.location,
						SubscriptionResourceGroupName:     &tt.testParams.resourceGroups.subscriptionRGName,
						GlobalResourceGroupName:           &tt.testParams.resourceGroups.globalResourceGroup,
						ACRLocationOverride:               &tt.testParams.overrideLocation,
						ACRReplicaDisabled:                &tt.testParams.acrReplicaDisabled,
						GlobalDevopsManagedIdentity:       &tt.testParams.globalDevopsManagedIdentity,
					},
					RPResourceGroupName:      tt.testParams.resourceGroups.rpResourceGroupName,
					GatewayResourceGroupName: tt.testParams.resourceGroups.gatewayResourceGroupName,
					Location:                 tt.testParams.location,
				},
				serviceKeyvault: mockKV,
				portalKeyvault:  mockKV,
				vmss:            mockVMSS,
				vmssvms:         mockVMSSVM,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, mockResourceGroups, mockMSIs, mockKV, mockVMSS, mockVMSSVM, tt.testParams)
			}

			err := d.PreDeploy(ctx, 0)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobalSubscription(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		location string
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	createOrUpdateAtSubscriptionScopeAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, "rp-global-subscription-"+tp.location, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name:       "Don't continue if deployment fails with error other than DeploymentFailed",
			testParams: testParams{location: location},
			mocks:      []mock{createOrUpdateAtSubscriptionScopeAndWaitMock(errGeneric)},
			wantErr:    "generic error",
		},
		{
			name:       "Don't continue if deployment fails with error DeploymentFailed five times",
			testParams: testParams{location: location},
			mocks:      []mock{createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError), createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError), createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError), createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError), createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError)},
			wantErr:    `Code="DeploymentFailed" Message="" Details=[{}]`,
		},
		{
			name:       "Pass successfully when deployment is successfulin second attempt",
			testParams: testParams{location: location},
			mocks:      []mock{createOrUpdateAtSubscriptionScopeAndWaitMock(deploymentFailedError), createOrUpdateAtSubscriptionScopeAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration: &Configuration{
						GlobalResourceGroupLocation: &tt.testParams.location,
					},
					Location: tt.testParams.location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployRPGlobalSubscription(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPSubscription(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		resourceGroup string
		location      string
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	CreateOrUpdateAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, tp.resourceGroup, "rp-production-subscription-"+tp.location, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "Don't continue if deployment fails",
			testParams: testParams{
				location:      location,
				resourceGroup: subscriptionRGName,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Pass successfully when deployment is successful",
			testParams: testParams{
				location:      location,
				resourceGroup: subscriptionRGName,
			},
			mocks: []mock{CreateOrUpdateAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration: &Configuration{
						SubscriptionResourceGroupName: &tt.testParams.resourceGroup,
					},
					Location: tt.testParams.location,
				},
				deployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployRPSubscription(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployManagedIdentity(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		resourceGroup      string
		deploymentFileName string
		deploymentName     string
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	CreateOrUpdateAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, tp.resourceGroup, tp.deploymentName, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "Don't continue if deployment file does not exist",
			testParams: testParams{
				deploymentFileName: notExistingFileName,
			},
			wantErr: "open " + notExistingFileName + ": file does not exist",
		},
		{
			name: "Don't continue if deployment fails",
			testParams: testParams{
				deploymentFileName: existingFileName,
				deploymentName:     existingFileDeploymentName,
				resourceGroup:      rgName,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Pass successfully when deployment is successful",
			testParams: testParams{
				deploymentFileName: existingFileName,
				deploymentName:     existingFileDeploymentName,
				resourceGroup:      rgName,
			},
			mocks: []mock{CreateOrUpdateAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration: &Configuration{},
				},
				deployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployManagedIdentity(ctx, tt.testParams.resourceGroup, tt.testParams.deploymentFileName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobal(t *testing.T) {
	ctx := context.Background()
	rpSPID := "rpSPIDTest"
	gwySPID := "gwySPIDTest"
	devopsSPID := "devopsSPIDTest"

	type testParams struct {
		resourceGroup string
		location      string
		rpSPID        string
		gwySPID       string
		devopsSPID    string
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	CreateOrUpdateAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, tp.resourceGroup, "rp-global-"+tp.location, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "Don't continue if deployment fails with error other than DeploymentFailed",
			testParams: testParams{
				location:      location,
				resourceGroup: globalRGName,
				rpSPID:        rpSPID,
				gwySPID:       gwySPID,
				devopsSPID:    devopsSPID,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Don't continue if deployment fails with DeploymentFailed error twice",
			testParams: testParams{
				location:      location,
				resourceGroup: globalRGName,
				rpSPID:        rpSPID,
				gwySPID:       gwySPID,
				devopsSPID:    devopsSPID,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(deploymentFailedError), CreateOrUpdateAndWaitMock(deploymentFailedError)},
			wantErr: `Code="DeploymentFailed" Message="" Details=[{}]`,
		},
		{
			name: "Pass successfully when deployment is successful in second attempt",
			testParams: testParams{
				location:      location,
				resourceGroup: globalRGName,
				rpSPID:        rpSPID,
				gwySPID:       gwySPID,
				devopsSPID:    devopsSPID,
			},
			mocks: []mock{CreateOrUpdateAndWaitMock(deploymentFailedError), CreateOrUpdateAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration: &Configuration{
						GlobalResourceGroupName: to.StringPtr(tt.testParams.resourceGroup),
					},
					Location: tt.testParams.location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployRPGlobal(ctx, tt.testParams.rpSPID, tt.testParams.gwySPID, tt.testParams.devopsSPID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobalACRReplication(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		resourceGroup string
		location      string
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	CreateOrUpdateAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, tp.resourceGroup, "rp-global-acr-replication-"+tp.location, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "Don't continue if deployment fails",
			testParams: testParams{
				location:      location,
				resourceGroup: globalRGName,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Pass when deployment is successful",
			testParams: testParams{
				location:      location,
				resourceGroup: globalRGName,
			},
			mocks: []mock{CreateOrUpdateAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration: &Configuration{
						GlobalResourceGroupName: to.StringPtr(tt.testParams.resourceGroup),
					},
					Location: tt.testParams.location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployRPGlobalACRReplication(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployPreDeploy(t *testing.T) {
	ctx := context.Background()
	spIDName := "testSPIDName"
	spID := "testSPID"

	type testParams struct {
		resourceGroup      string
		deploymentFileName string
		deploymentName     string
		spIDName           string
		spID               string
		isCreate           bool
	}
	type mock func(*mock_features.MockDeploymentsClient, testParams)
	CreateOrUpdateAndWaitMock := func(returnError error) mock {
		return func(d *mock_features.MockDeploymentsClient, tp testParams) {
			d.EXPECT().CreateOrUpdateAndWait(ctx, tp.resourceGroup, tp.deploymentName, gomock.Any()).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "Don't continue if deployment file does not exist",
			testParams: testParams{
				resourceGroup:      rgName,
				deploymentFileName: notExistingFileName,
				spIDName:           spIDName,
				spID:               spID,
			},
			wantErr: "open " + notExistingFileName + ": file does not exist",
		},
		{
			name: "Don't continue if deployment fails",
			testParams: testParams{
				resourceGroup:      rgName,
				deploymentFileName: existingFileName,
				deploymentName:     existingFileDeploymentName,
				spIDName:           spIDName,
				spID:               spID,
			},
			mocks:   []mock{CreateOrUpdateAndWaitMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Pass when deployment is successful",
			testParams: testParams{
				resourceGroup:      rgName,
				deploymentFileName: existingFileName,
				deploymentName:     existingFileDeploymentName,
				spIDName:           spIDName,
				spID:               spID,
			},
			mocks: []mock{CreateOrUpdateAndWaitMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					Configuration:            &Configuration{},
					GatewayResourceGroupName: tt.testParams.resourceGroup,
				},
				deployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, tt.testParams)
			}

			err := d.deployPreDeploy(ctx, tt.testParams.resourceGroup, tt.testParams.deploymentFileName, tt.testParams.spIDName, tt.testParams.spID, tt.testParams.isCreate)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestConfigureServiceSecrets(t *testing.T) {
	ctx := context.Background()
	oneMissingSecrets := []string{env.FrontendEncryptionSecretV2Name, env.PortalServerSessionKeySecretName, env.EncryptionSecretName, env.FrontendEncryptionSecretName, env.PortalServerSSHKeySecretName}
	oneMissingSecretItems := []*azsecretssdk.SecretProperties{}
	for _, secret := range oneMissingSecrets {
		oneMissingSecretItems = append(oneMissingSecretItems, &azsecretssdk.SecretProperties{ID: pointerutils.ToPtr(azsecretssdk.ID("https://myvaultname.vault.azure.net/keys/" + secret + "/whatever"))})
	}
	allSecrets := []string{env.EncryptionSecretV2Name, env.FrontendEncryptionSecretV2Name, env.PortalServerSessionKeySecretName, env.EncryptionSecretName, env.FrontendEncryptionSecretName, env.PortalServerSSHKeySecretName}
	allSecretItems := []*azsecretssdk.SecretProperties{}
	for _, secret := range allSecrets {
		allSecretItems = append(allSecretItems, &azsecretssdk.SecretProperties{ID: pointerutils.ToPtr(azsecretssdk.ID("https://myvaultname.vault.azure.net/keys/" + secret + "/whatever"))})
	}

	type testParams struct {
		vmssName      string
		instanceID    string
		resourceGroup string
		restartScript string
	}
	type mock func(*mock_azsecrets.MockClient, *mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	getSecretsMock := func(items []*azsecretssdk.SecretProperties, returnError error) mock {
		return func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			k.EXPECT().NewListSecretPropertiesPager(nil).Return(mock_azuresdk.NewPager(listForItems(items), []error{returnError}))
		}
	}
	getSecretMock := func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		k.EXPECT().GetSecret(ctx, gomock.Any(), "", nil).Return(newSecretBundle, nil)
	}
	setSecretMock := func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, nil)
	}
	vmssListMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			vmss.EXPECT().List(ctx, tp.resourceGroup).Return(vmsss, returnError).AnyTimes()
		}
	}
	vmssVMsListMock := func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().List(ctx, tp.resourceGroup, tp.vmssName, "", "", "").Return(vms, nil).AnyTimes()
	}
	vmRestartMock := func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, tp.resourceGroup, tp.vmssName, tp.instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{tp.restartScript},
		}).Return(nil).AnyTimes()
	}
	instanceViewMock := func(k *mock_azsecrets.MockClient, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), tp.resourceGroup, tp.vmssName, tp.instanceID).Return(healthyVMSS, nil).AnyTimes()
	}

	for _, tt := range []struct {
		name         string
		secretToFind string
		testParams   testParams
		mocks        []mock
		wantErr      string
	}{
		{
			name: "return error if ensureAndRotateSecret fails",
			mocks: []mock{
				getSecretsMock(allSecretItems, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret passes without rotating any secret but ensureSecret fails",
			mocks: []mock{
				getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret passes with rotating a missing secret but ensureSecret fails",
			mocks: []mock{
				getSecretsMock(oneMissingSecretItems, nil), setSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret, ensureSecret passes without rotating a secret but ensureSecretKey fails",
			mocks: []mock{
				getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "return nil if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes without rotating a secret",
			mocks: []mock{
				getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil),
			},
		},
		{
			name: "return error if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes with rotating secret in each ensure function call but restartoldscaleset failing",
			testParams: testParams{
				vmssName:      vmssName,
				instanceID:    instanceID,
				resourceGroup: rgName,
			},
			mocks: []mock{
				getSecretsMock(oneMissingSecretItems, nil), setSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), vmssListMock(errGeneric),
			},
			wantErr: "generic error",
		},
		{
			name: "return nil if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes with rotating secret and restartoldscaleset passess successfully",
			testParams: testParams{
				vmssName:      vmssName,
				instanceID:    instanceID,
				resourceGroup: rgName,
				restartScript: rpRestartScript,
			},
			mocks: []mock{
				getSecretsMock(oneMissingSecretItems, nil), setSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretMock, getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), getSecretsMock(allSecretItems, nil), vmssListMock(nil), vmssVMsListMock, vmRestartMock, instanceViewMock,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)
			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			mockVMSSVM := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					RPResourceGroupName:      tt.testParams.resourceGroup,
					GatewayResourceGroupName: tt.testParams.resourceGroup,
				},
				serviceKeyvault: mockKV,
				portalKeyvault:  mockKV,
				vmss:            mockVMSS,
				vmssvms:         mockVMSSVM,
			}

			for _, m := range tt.mocks {
				m(mockKV, mockVMSS, mockVMSSVM, tt.testParams)
			}

			err := d.configureServiceSecrets(ctx, 0)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureAndRotateSecret(t *testing.T) {
	ctx := context.Background()
	oldSecretBundle := azsecretssdk.GetSecretResponse{
		Secret: azsecretssdk.Secret{
			Attributes: &azsecretssdk.SecretAttributes{Created: pointerutils.ToPtr(time.Now().Add(-rotateSecretAfter))},
		},
	}

	type testParams struct {
		secretToFind string
	}
	type mock func(*mock_azsecrets.MockClient, testParams)
	getSecretsMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().NewListSecretPropertiesPager(nil).Return(secretItems(returnError))
		}
	}
	getSecretMock := func(secretBundle azsecretssdk.GetSecretResponse, returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().GetSecret(ctx, tp.secretToFind, "", nil).Return(secretBundle, returnError)
		}
	}
	setSecretMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().SetSecret(ctx, tp.secretToFind, gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
		wantBool   bool
	}{
		{
			name:       "return false and error if GetSecrets fails",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(errGeneric)},
			wantBool:   false,
			wantErr:    "generic error",
		},
		{
			name:       "return false and error if GetSecrets passes but GetSecret fails for the found secret",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil), getSecretMock(newSecretBundle, errGeneric)},
			wantBool:   false,
			wantErr:    "generic error",
		},
		{
			name:       "return false and nil if GetSecrets and GetSecret passes and the secret is not too old",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil), getSecretMock(newSecretBundle, nil)},
			wantBool:   false,
		},
		{
			name:       "return true and error if GetSecrets & GetSecret passes and the secret is old but new secret creation fails",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil), getSecretMock(oldSecretBundle, nil), setSecretMock(errGeneric)},
			wantBool:   true,
			wantErr:    "generic error",
		},
		{
			name:       "return true and nil if GetSecrets & GetSecret passes and the secret is old and new secret creation passes",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil), getSecretMock(oldSecretBundle, nil), setSecretMock(nil)},
			wantBool:   true,
		},
		{
			name:       "return true and nil if the secret is not present and new secret creation passes",
			testParams: testParams{secretToFind: noSecretExists},
			mocks:      []mock{getSecretsMock(nil), setSecretMock(nil)},
			wantBool:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV, tt.testParams)
			}

			got, err := d.ensureAndRotateSecret(ctx, mockKV, tt.testParams.secretToFind, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestEnsureSecret(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		secretToFind string
	}
	type mock func(*mock_azsecrets.MockClient, testParams)
	getSecretsMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().NewListSecretPropertiesPager(nil).Return(secretItems(returnError))
		}
	}
	setSecretMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().SetSecret(ctx, tp.secretToFind, gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
		wantBool   bool
	}{
		{
			name:       "return false and error if GetSecrets fails",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(errGeneric)},
			wantBool:   false,
			wantErr:    "generic error",
		},
		{
			name:       "return false and nil if GetSecrets passes and secret is found",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil)},
			wantBool:   false,
		},
		{
			name:       "return true and error if GetSecrets passes but secret is not found and new secret creation fails",
			testParams: testParams{secretToFind: noSecretExists},
			mocks:      []mock{getSecretsMock(nil), setSecretMock(errGeneric)},
			wantBool:   true,
			wantErr:    "generic error",
		},
		{
			name:       "return true and nil if GetSecrets passes but secret is not found and new secret creation also passes",
			testParams: testParams{secretToFind: noSecretExists},
			mocks:      []mock{getSecretsMock(nil), setSecretMock(nil)},
			wantBool:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV, tt.testParams)
			}

			got, err := d.ensureSecret(ctx, mockKV, tt.testParams.secretToFind, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestCreateSecret(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		secretToCreate string
	}
	type mock func(*mock_azsecrets.MockClient, testParams)
	setSecretMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().SetSecret(ctx, tp.secretToCreate, gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "return error if new secret creation fails",
			testParams: testParams{
				secretToCreate: noSecretExists,
			},
			mocks:   []mock{setSecretMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "return nil new secret creation passes",
			testParams: testParams{
				secretToCreate: noSecretExists,
			},
			mocks: []mock{setSecretMock(nil)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV, tt.testParams)
			}

			err := d.createSecret(ctx, mockKV, tt.testParams.secretToCreate, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureSecretKey(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		secretToFind string
	}
	type mock func(*mock_azsecrets.MockClient, testParams)
	getSecretsMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().NewListSecretPropertiesPager(nil).Return(secretItems(returnError))
		}
	}
	setSecretMock := func(returnError error) mock {
		return func(k *mock_azsecrets.MockClient, tp testParams) {
			k.EXPECT().SetSecret(ctx, tp.secretToFind, gomock.Any(), nil).Return(azsecretssdk.SetSecretResponse{}, returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
		wantBool   bool
	}{
		{
			name:       "return false and error if GetSecrets fails",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(errGeneric)},
			wantBool:   false,
			wantErr:    "generic error",
		},
		{
			name:       "return false and nil if GetSecrets passes and secret is found",
			testParams: testParams{secretToFind: secretExists},
			mocks:      []mock{getSecretsMock(nil)},
			wantBool:   false,
		},
		{
			name:       "return true and error if GetSecrets passes but secret is not found and new secret creation fails",
			testParams: testParams{secretToFind: noSecretExists},
			mocks:      []mock{getSecretsMock(nil), setSecretMock(errGeneric)},
			wantBool:   true,
			wantErr:    "generic error",
		},
		{
			name:       "return true and nil if GetSecrets passes but secret is not found and new secret creation also passes",
			testParams: testParams{secretToFind: noSecretExists},
			mocks:      []mock{getSecretsMock(nil), setSecretMock(nil)},
			wantBool:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV, tt.testParams)
			}

			got, err := d.ensureSecretKey(ctx, mockKV, tt.testParams.secretToFind)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestRestartOldScalesets(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		resourceGroup string
		vmssName      string
		instanceID    string
		restartScript string
	}
	type mock func(*mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	listVMSSMock := func(returnVMSS []mgmtcompute.VirtualMachineScaleSet, returnError error) mock {
		return func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			vmss.EXPECT().List(ctx, tp.resourceGroup).Return(returnVMSS, returnError)
		}
	}
	listVMSSVMMock := func(returnError error) mock {
		return func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			vmssvms.EXPECT().List(ctx, tp.resourceGroup, tp.vmssName, "", "", "").Return(vms, returnError)
		}
	}
	vmRestartMock := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, tp.resourceGroup, tp.vmssName, tp.instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{tp.restartScript},
		}).Return(nil)
	}
	getInstanceViewMock := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), tp.resourceGroup, tp.vmssName, tp.instanceID).Return(healthyVMSS, nil)
	}

	for _, tt := range []struct {
		name       string
		mocks      []mock
		testParams testParams
		wantErr    string
	}{
		{
			name:       "Don't continue if vmss list fails",
			testParams: testParams{resourceGroup: rgName},
			mocks:      []mock{listVMSSMock(vmsss, errGeneric)},
			wantErr:    "generic error",
		},
		{
			name:       "Don't continue if vmss list has an invalid vmss name",
			testParams: testParams{resourceGroup: rgName},
			mocks:      []mock{listVMSSMock(invalidVMSSs, nil)},
			wantErr:    "400: InvalidResource: : provided vmss other-vmss does not match RP prefix",
		},
		{
			name: "Don't continue if vmssvms list fails",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
			},
			mocks:   []mock{listVMSSMock(vmsss, nil), listVMSSVMMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "Restart is successful for the VMs in VMSS",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				instanceID:    instanceID,
				restartScript: rpRestartScript,
			},
			mocks: []mock{listVMSSMock(vmsss, nil), listVMSSVMMock(nil), vmRestartMock, getInstanceViewMock},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			mockVMSSVM := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				vmss:    mockVMSS,
				vmssvms: mockVMSSVM,
				config: &RPConfig{
					RPResourceGroupName: rgName,
				},
			}

			for _, m := range tt.mocks {
				m(mockVMSS, mockVMSSVM, tt.testParams)
			}

			err := d.restartOldScalesets(ctx, 0)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestRestartOldScaleset(t *testing.T) {
	ctx := context.Background()

	type testParams struct {
		resourceGroup string
		vmssName      string
		instanceID    string
		restartScript string
	}
	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	getInstanceViewMock := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
		c.EXPECT().GetInstanceView(gomock.Any(), tp.resourceGroup, tp.vmssName, tp.instanceID).Return(healthyVMSS, nil)
	}
	listVMSSVMMock := func(returnError error) mock {
		return func(c *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			c.EXPECT().List(ctx, tp.resourceGroup, tp.vmssName, "", "", "").Return(vms, returnError)
		}
	}
	vmRestartMock := func(returnError error) mock {
		return func(c *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			c.EXPECT().RunCommandAndWait(ctx, tp.resourceGroup, tp.vmssName, tp.instanceID, mgmtcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
				Script:    &[]string{tp.restartScript},
			}).Return(returnError)
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name:       "Return an error if the VMSS is not RP",
			testParams: testParams{vmssName: invalidVMSSName},
			wantErr:    "400: InvalidResource: : provided vmss other-vmss does not match RP prefix",
		},
		{
			name: "list VMSS failed",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				instanceID:    instanceID,
			},
			mocks:   []mock{listVMSSVMMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "rp restart script failed",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				instanceID:    instanceID,
				restartScript: rpRestartScript,
			},
			mocks:   []mock{listVMSSVMMock(nil), vmRestartMock(errGeneric)},
			wantErr: "generic error",
		},
		{
			name: "restart script passes and wait for readiness is successful",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				instanceID:    instanceID,
				restartScript: rpRestartScript,
			},
			mocks: []mock{listVMSSVMMock(nil), vmRestartMock(nil), getInstanceViewMock},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				vmssvms: mockVMSS,
				config: &RPConfig{
					RPResourceGroupName: rgName,
				},
			}

			for _, m := range tt.mocks {
				m(mockVMSS, tt.testParams)
			}

			err := d.restartOldScaleset(ctx, tt.testParams.vmssName, 0)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestWaitForReadiness(t *testing.T) {
	ctxRegular, cancelRegular := context.WithTimeout(context.Background(), 11*time.Second)
	ctxFastTimeout, cancelFastTimeout := context.WithTimeout(context.Background(), 0*time.Second)

	type testParams struct {
		resourceGroup string
		vmssName      string
		vmInstanceID  string
		ctx           context.Context
		cancel        context.CancelFunc
	}
	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	getInstanceViewMock := func(vm mgmtcompute.VirtualMachineScaleSetVMInstanceView) mock {
		return func(c *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			c.EXPECT().GetInstanceView(tp.ctx, tp.resourceGroup, tp.vmssName, tp.vmInstanceID).Return(vm, nil).AnyTimes()
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantErr    string
	}{
		{
			name: "fail after context times out",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				vmInstanceID:  instanceID,
				ctx:           ctxFastTimeout,
				cancel:        cancelFastTimeout,
			},
			mocks:   []mock{getInstanceViewMock(unhealthyVMSS)},
			wantErr: "context deadline exceeded",
		},
		{
			name: "run successfully after confirming healthy status",
			testParams: testParams{
				resourceGroup: rgName,
				vmssName:      vmssName,
				vmInstanceID:  instanceID,
				ctx:           ctxRegular,
				cancel:        cancelRegular,
			},
			mocks: []mock{getInstanceViewMock(healthyVMSS)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				vmssvms: mockVMSS,
				config: &RPConfig{
					RPResourceGroupName: rgName,
				},
			}

			for _, m := range tt.mocks {
				m(mockVMSS, tt.testParams)
			}

			defer tt.testParams.cancel()
			err := d.waitForReadiness(tt.testParams.ctx, tt.testParams.vmssName, tt.testParams.vmInstanceID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestIsVMInstanceHealthy(t *testing.T) {
	ctx := context.Background()
	rpRGName := "testRPRG"
	gatewayRGName := "testGatewayRG"

	type testParams struct {
		resourceGroup string
		vmssName      string
		instanceID    string
	}
	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient, testParams)
	getInstanceViewMock := func(vm mgmtcompute.VirtualMachineScaleSetVMInstanceView, returnError error) mock {
		return func(c *mock_compute.MockVirtualMachineScaleSetVMsClient, tp testParams) {
			c.EXPECT().GetInstanceView(ctx, tp.resourceGroup, tp.vmssName, tp.instanceID).Return(vm, returnError).AnyTimes()
		}
	}

	for _, tt := range []struct {
		name       string
		testParams testParams
		mocks      []mock
		wantBool   bool
	}{
		{
			name: "return false if GetInstanceView failed for RP resource group",
			testParams: testParams{
				resourceGroup: rpRGName,
				vmssName:      vmssName,
				instanceID:    instanceID,
			},
			mocks:    []mock{getInstanceViewMock(healthyVMSS, errGeneric)},
			wantBool: false,
		},
		{
			name: "return false if GetInstanceView failed for Gateway resource group",
			testParams: testParams{
				resourceGroup: gatewayRGName,
				vmssName:      vmssName,
				instanceID:    instanceID,
			},
			mocks:    []mock{getInstanceViewMock(healthyVMSS, errGeneric)},
			wantBool: false,
		},
		{
			name: "return false if GetInstanceView return unhealthy VM",
			testParams: testParams{
				resourceGroup: rpRGName,
				vmssName:      vmssName,
				instanceID:    instanceID,
			},
			mocks:    []mock{getInstanceViewMock(unhealthyVMSS, nil)},
			wantBool: false,
		},
		{
			name: "return true if GetInstanceView return healthy VM",
			testParams: testParams{
				resourceGroup: rpRGName,
				vmssName:      vmssName,
				instanceID:    instanceID,
			},
			mocks:    []mock{getInstanceViewMock(healthyVMSS, nil)},
			wantBool: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				vmssvms: mockVMSS,
			}

			for _, m := range tt.mocks {
				m(mockVMSS, tt.testParams)
			}

			got := d.isVMInstanceHealthy(ctx, tt.testParams.resourceGroup, tt.testParams.vmssName, tt.testParams.instanceID)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}
