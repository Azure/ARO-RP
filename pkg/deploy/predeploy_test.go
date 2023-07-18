package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	gofrsuuid "github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_msi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/msi"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestPreDeploy(t *testing.T) {
	ctx := context.Background()
	subscriptionRgName := "testRG-subscription"
	globalRgName := "testRG-global"
	rpRgName := "testRG-aro-rp"
	gatewayRgName := "testRG-gwy"
	location := "testLocation"
	overrideLocation := "overrideTestLocation"
	group := mgmtfeatures.ResourceGroup{
		Location: &location,
	}
	fakeMSIObjectId, _ := gofrsuuid.NewV4()
	msi := mgmtmsi.Identity{
		UserAssignedIdentityProperties: &mgmtmsi.UserAssignedIdentityProperties{
			PrincipalID: &fakeMSIObjectId,
		},
	}
	deployment := mgmtfeatures.DeploymentExtended{}
	partialSecretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretV2Name),
		},
	}
	rpVMSSName := rpVMSSPrefix + "test"
	nowUnixTime := date.NewUnixTimeFromSeconds(float64(time.Now().Unix()))
	newSecretBundle := azkeyvault.SecretBundle{
		Attributes: &azkeyvault.SecretAttributes{
			Created: &nowUnixTime,
		},
	}
	vmsss := []mgmtcompute.VirtualMachineScaleSet{
		{
			Name: to.StringPtr(rpVMSSName),
		},
	}
	allSecretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.PortalServerSessionKeySecretName),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretName),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretName),
		},
		{
			ID: to.StringPtr(env.PortalServerSSHKeySecretName),
		},
	}
	instanceID := "testID"
	vms := []mgmtcompute.VirtualMachineScaleSetVM{
		{
			InstanceID: to.StringPtr(instanceID),
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}

	type mock func(*mock_features.MockDeploymentsClient, *mock_features.MockResourceGroupsClient, *mock_msi.MockUserAssignedIdentitiesClient, *mock_keyvault.MockManager, *mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient)
	genericSubscriptionDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		).AnyTimes()
	}
	subscriptionDeploymentSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, gomock.Any(), gomock.Any()).Return(nil)
	}
	subscriptionRGDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, subscriptionRgName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	globalRGDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRgName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	gatewayRGDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, gatewayRgName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	rpRGDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rpRgName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	}
	subscriptionResourceGroupDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		rg.EXPECT().CreateOrUpdate(ctx, subscriptionRgName, gomock.Any()).Return(
			group,
			errors.New("generic error"),
		)
	}
	globalResourceGroupDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		rg.EXPECT().CreateOrUpdate(ctx, globalRgName, gomock.Any()).Return(
			group,
			errors.New("generic error"),
		)
	}
	rpResourceGroupDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		rg.EXPECT().CreateOrUpdate(ctx, rpRgName, gomock.Any()).Return(
			group,
			errors.New("generic error"),
		)
	}
	gatewayResourceGroupDeploymentFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		rg.EXPECT().CreateOrUpdate(ctx, gatewayRgName, gomock.Any()).Return(
			group,
			errors.New("generic error"),
		)
	}
	resourceGroupDeploymentSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		rg.EXPECT().CreateOrUpdate(ctx, gomock.Any(), gomock.Any()).Return(group, nil)
	}
	rpMSIGetFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		m.EXPECT().Get(ctx, rpRgName, gomock.Any()).Return(msi, errors.New("generic error"))
	}
	rpMSIGetSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		m.EXPECT().Get(ctx, rpRgName, gomock.Any()).Return(msi, nil)
	}
	gatewayMSIGetFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		m.EXPECT().Get(ctx, gatewayRgName, gomock.Any()).Return(msi, errors.New("generic error"))
	}
	gatewayMSIGetSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		m.EXPECT().Get(ctx, gatewayRgName, gomock.Any()).Return(msi, nil)
	}
	getDeploymentFailedWithDeploymentNotFound := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		d.EXPECT().Get(ctx, gatewayRgName, gomock.Any()).Return(deployment, autorest.DetailedError{
			Original: &azure.RequestError{
				ServiceError: &azure.ServiceError{
					Code: "DeploymentNotFound",
					Details: []map[string]interface{}{
						{},
					},
				},
			},
		})
	}
	getSecretsFailed := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			partialSecretItems, errors.New("generic error"),
		)
	}
	getSecretsSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			allSecretItems, nil,
		)
	}
	getNewSecretSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecret(ctx, gomock.Any()).Return(
			newSecretBundle, nil,
		)
	}
	getPartialSecretsSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			partialSecretItems, nil,
		)
	}
	setSecretSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}
	vmssListSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, gomock.Any()).Return(
			vmsss, nil,
		)
	}
	vmssVMsListSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().List(ctx, gomock.Any(), gomock.Any(), "", "", "").Return(
			vms, nil,
		)
	}
	restartSuccessful := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, gomock.Any(), gomock.Any(), instanceID, gomock.Any()).Return(nil)
	}
	healthyInstanceView := func(d *mock_features.MockDeploymentsClient, rg *mock_features.MockResourceGroupsClient, m *mock_msi.MockUserAssignedIdentitiesClient, k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), gomock.Any(), gomock.Any(), instanceID).Return(healthyVMSS, nil)
	}

	for _, tt := range []struct {
		name                     string
		location                 string
		overrideLocation         string
		acrReplicaDisabled       bool
		subscriptionRgName       string
		globalResourceGroup      string
		rpResourceGroupName      string
		gatewayResourceGroupName string
		mocks                    []mock
		wantErr                  string
	}{
		{
			name:     "don't continue if Global Subscription RBAC DeploymentFailed",
			location: location,
			mocks: []mock{
				genericSubscriptionDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:               "don't continue if Global Subscription RBAC Deployment is Successful but SubscriptionResourceGroup creation fails",
			location:           location,
			subscriptionRgName: subscriptionRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, subscriptionResourceGroupDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                "don't continue if Global Subscription RBAC Deployment is Successful but GlobalResourceGroup creation fails",
			location:            location,
			subscriptionRgName:  subscriptionRgName,
			globalResourceGroup: globalRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, globalResourceGroupDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                "don't continue if Global Subscription RBAC Deployment is Successful but RPResourceGroup creation fails",
			location:            location,
			subscriptionRgName:  subscriptionRgName,
			globalResourceGroup: globalRgName,
			rpResourceGroupName: rpRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, rpResourceGroupDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment is successful but GatewayResourceGroup creation fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, gatewayResourceGroupDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment & resource group creation is successful but rp-subscription template deployment fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, subscriptionRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment, resource group creation and rp-subscription template deployment is successful but rp managed identity get fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment, resource group creation and rp-subscription template deployment is successful but gateway managed identity get fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment, resource group creation and rp-subscription template deployment, msi get is successful but rpglobal deployment get fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, globalRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment, resource group creation and rp-subscription template deployment, msi get is successful but rpglobal deployment get fails",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, globalRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if Global Subscription RBAC Deployment, resource group creation, rp-subscription deployment, rpglobal deployment is successful but ACR Replication fails",
			location:                 location,
			overrideLocation:         overrideLocation,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, globalRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if skipping ACR Replication due to no ACRLocationOverride but failing gateway predeploy",
			location:                 location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, gatewayRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if skipping ACR Replication due to same ACRLocationOverride as location but failing gateway predeploy",
			location:                 location,
			overrideLocation:         location,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, gatewayRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue if skipping ACR Replication due to ACRReplicaDisabled but failing gateway predeploy",
			location:                 location,
			overrideLocation:         overrideLocation,
			acrReplicaDisabled:       true,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, gatewayRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "don't continue gateway predeploy is successful but rp predeploy failed",
			location:                 location,
			overrideLocation:         overrideLocation,
			acrReplicaDisabled:       true,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, deploymentSuccessful, rpRGDeploymentFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "get error for the configureServiceSecrets",
			location:                 location,
			overrideLocation:         overrideLocation,
			acrReplicaDisabled:       true,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, deploymentSuccessful, deploymentSuccessful, getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name:                     "Everything is successful",
			location:                 location,
			overrideLocation:         overrideLocation,
			acrReplicaDisabled:       true,
			subscriptionRgName:       subscriptionRgName,
			globalResourceGroup:      globalRgName,
			rpResourceGroupName:      rpRgName,
			gatewayResourceGroupName: gatewayRgName,
			mocks: []mock{
				subscriptionDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, resourceGroupDeploymentSuccessful, deploymentSuccessful, deploymentSuccessful, rpMSIGetSuccessful, deploymentSuccessful, gatewayMSIGetSuccessful, deploymentSuccessful, getDeploymentFailedWithDeploymentNotFound, deploymentSuccessful, deploymentSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getSecretsSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, vmssListSuccessful, vmssVMsListSuccessful, restartSuccessful, healthyInstanceView, vmssListSuccessful, vmssVMsListSuccessful, restartSuccessful, healthyInstanceView,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)
			mockResourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			mockMSIs := mock_msi.NewMockUserAssignedIdentitiesClient(controller)
			mockKV := mock_keyvault.NewMockManager(controller)
			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			mockVMSSVM := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log:                    logrus.NewEntry(logrus.StandardLogger()),
				globaldeployments:      mockDeployments,
				deployments:            mockDeployments,
				groups:                 mockResourceGroups,
				globalgroups:           mockResourceGroups,
				userassignedidentities: mockMSIs,
				config: &RPConfig{
					Configuration: &Configuration{
						GlobalResourceGroupLocation:       &tt.location,
						SubscriptionResourceGroupLocation: &tt.location,
						SubscriptionResourceGroupName:     &tt.subscriptionRgName,
						GlobalResourceGroupName:           &tt.globalResourceGroup,
						ACRLocationOverride:               &tt.overrideLocation,
						ACRReplicaDisabled:                &tt.acrReplicaDisabled,
					},
					RPResourceGroupName:      tt.rpResourceGroupName,
					GatewayResourceGroupName: tt.gatewayResourceGroupName,
					Location:                 tt.location,
				},
				serviceKeyvault: mockKV,
				portalKeyvault:  mockKV,
				vmss:            mockVMSS,
				vmssvms:         mockVMSSVM,
			}

			for _, m := range tt.mocks {
				m(mockDeployments, mockResourceGroups, mockMSIs, mockKV, mockVMSS, mockVMSSVM)
			}

			err := d.PreDeploy(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobalSubscription(t *testing.T) {
	ctx := context.Background()
	location := "locationTest"

	type mock func(*mock_features.MockDeploymentsClient)
	subscriptionDeploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		).AnyTimes()
	}
	subscriptionDeploymentFailedWithDeploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, gomock.Any(), gomock.Any()).Return(
			&azure.ServiceError{
				Code: "DeploymentFailed",
				Details: []map[string]interface{}{
					{},
				},
			},
		)
	}
	subscriptionDeploymentSuccessful := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAtSubscriptionScopeAndWait(ctx, gomock.Any(), gomock.Any()).Return(nil)
	}

	for _, tt := range []struct {
		name               string
		deploymentFileName string
		mocks              []mock
		wantErr            string
	}{
		{
			name:    "Don't continue if deployment fails with error other than DeploymentFailed",
			mocks:   []mock{subscriptionDeploymentFailed},
			wantErr: "generic error",
		},
		{
			name:    "Don't continue if deployment fails with error DeploymentFailed five times",
			mocks:   []mock{subscriptionDeploymentFailedWithDeploymentFailed, subscriptionDeploymentFailedWithDeploymentFailed, subscriptionDeploymentFailedWithDeploymentFailed, subscriptionDeploymentFailedWithDeploymentFailed, subscriptionDeploymentFailedWithDeploymentFailed},
			wantErr: `Code="DeploymentFailed" Message="" Details=[{}]`,
		},
		{
			name:  "Pass successfully when deployment is successfulin second attempt",
			mocks: []mock{subscriptionDeploymentFailedWithDeploymentFailed, subscriptionDeploymentSuccessful},
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
						GlobalResourceGroupLocation: &location,
					},
					Location: location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments)
			}

			err := d.deployRPGlobalSubscription(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPSubscription(t *testing.T) {
	ctx := context.Background()
	location := "locationTest"
	subscriptionRGName := "rgTest"

	type mock func(*mock_features.MockDeploymentsClient)
	deploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, subscriptionRGName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentSuccess := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, subscriptionRGName, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name               string
		deploymentFileName string
		mocks              []mock
		wantErr            string
	}{
		{
			name:    "Don't continue if deployment fails",
			mocks:   []mock{deploymentFailed},
			wantErr: "generic error",
		},
		{
			name:  "Pass successfully when deployment is successful",
			mocks: []mock{deploymentSuccess},
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
						SubscriptionResourceGroupName: &subscriptionRGName,
					},
					Location: location,
				},
				deployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments)
			}

			err := d.deployRPSubscription(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployManagedIdentity(t *testing.T) {
	ctx := context.Background()
	rgName := "rgTest"
	existingFileName := generator.FileGatewayProductionPredeploy
	deploymentName := strings.TrimSuffix(existingFileName, ".json")
	notExistingFileName := "testFile"

	type mock func(*mock_features.MockDeploymentsClient)
	deploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentSuccess := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name               string
		deploymentFileName string
		mocks              []mock
		wantErr            string
	}{
		{
			name:               "Don't continue if deployment file does not exist",
			deploymentFileName: notExistingFileName,
			wantErr:            "open " + notExistingFileName + ": file does not exist",
		},
		{
			name:               "Don't continue if deployment fails",
			deploymentFileName: existingFileName,
			mocks:              []mock{deploymentFailed},
			wantErr:            "generic error",
		},
		{
			name:               "Pass successfully when deployment is successful",
			deploymentFileName: existingFileName,
			mocks:              []mock{deploymentSuccess},
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
				m(mockDeployments)
			}

			err := d.deployManagedIdentity(ctx, rgName, tt.deploymentFileName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobal(t *testing.T) {
	ctx := context.Background()
	location := "locationTest"
	globalRGName := "globalRGTest"
	rpSPID := "rpSPIDTest"
	gwySPID := "gwySPIDTest"

	type mock func(*mock_features.MockDeploymentsClient)
	deploymentFailedWithGenericError := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRGName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRGName, gomock.Any(), gomock.Any()).Return(
			&azure.ServiceError{
				Code: "DeploymentFailed",
				Details: []map[string]interface{}{
					{},
				},
			},
		)
	}
	deploymentSuccess := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRGName, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name    string
		mocks   []mock
		wantErr string
	}{
		{
			name:    "Don't continue if deployment fails with error other than DeploymentFailed",
			mocks:   []mock{deploymentFailedWithGenericError},
			wantErr: "generic error",
		},
		{
			name:    "Don't continue if deployment fails with DeploymentFailed error twice",
			mocks:   []mock{deploymentFailed, deploymentFailed},
			wantErr: `Code="DeploymentFailed" Message="" Details=[{}]`,
		},
		{
			name:  "Pass successfully when deployment is successful in second attempt",
			mocks: []mock{deploymentFailed, deploymentSuccess},
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
						GlobalResourceGroupName: to.StringPtr(globalRGName),
					},
					Location: location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments)
			}

			err := d.deployRPGlobal(ctx, rpSPID, gwySPID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployRPGlobalACRReplication(t *testing.T) {
	ctx := context.Background()
	globalRGName := "globalRGTest"
	location := "testLocation"

	type mock func(*mock_features.MockDeploymentsClient)
	deploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRGName, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentSuccess := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, globalRGName, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name    string
		mocks   []mock
		wantErr string
	}{
		{
			name:    "Don't continue if deployment fails",
			mocks:   []mock{deploymentFailed},
			wantErr: "generic error",
		},
		{
			name:  "Pass when deployment is successful",
			mocks: []mock{deploymentSuccess},
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
						GlobalResourceGroupName: to.StringPtr(globalRGName),
					},
					Location: location,
				},
				globaldeployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments)
			}

			err := d.deployRPGlobalACRReplication(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeployPreDeploy(t *testing.T) {
	ctx := context.Background()
	rgName := "testRG"
	gwyRGName := "testGwyRG"
	existingFileName := generator.FileGatewayProductionPredeploy
	deploymentName := strings.TrimSuffix(existingFileName, ".json")
	notExistingFileName := "testFile"
	spIDName := "testSPIDName"
	spID := "testSPID"

	type mock func(*mock_features.MockDeploymentsClient)
	deploymentFailed := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	deploymentSuccess := func(d *mock_features.MockDeploymentsClient) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name               string
		resourceGroupName  string
		deploymentFileName string
		spIDName           string
		spID               string
		isCreate           bool
		mocks              []mock
		wantErr            string
	}{
		{
			name:               "Don't continue if deployment file does not exist",
			resourceGroupName:  rgName,
			deploymentFileName: notExistingFileName,
			spIDName:           spIDName,
			spID:               spID,
			wantErr:            "open " + notExistingFileName + ": file does not exist",
		},
		{
			name:               "Don't continue if deployment fails",
			resourceGroupName:  rgName,
			deploymentFileName: existingFileName,
			spIDName:           spIDName,
			spID:               spID,
			mocks:              []mock{deploymentFailed},
			wantErr:            "generic error",
		},
		{
			name:               "Pass when deployment is successful",
			resourceGroupName:  rgName,
			deploymentFileName: existingFileName,
			spIDName:           spIDName,
			spID:               spID,
			mocks:              []mock{deploymentSuccess},
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
					GatewayResourceGroupName: gwyRGName,
				},
				deployments: mockDeployments,
			}

			for _, m := range tt.mocks {
				m(mockDeployments)
			}

			err := d.deployPreDeploy(ctx, tt.resourceGroupName, tt.deploymentFileName, tt.spIDName, tt.spID, tt.isCreate)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestConfigureServiceSecrets(t *testing.T) {
	ctx := context.Background()
	rpVMSSName := rpVMSSPrefix + "test"
	rgName := "rgTest"
	nowUnixTime := date.NewUnixTimeFromSeconds(float64(time.Now().Unix()))
	newSecretBundle := azkeyvault.SecretBundle{
		Attributes: &azkeyvault.SecretAttributes{
			Created: &nowUnixTime,
		},
	}
	vmsss := []mgmtcompute.VirtualMachineScaleSet{
		{
			Name: to.StringPtr(rpVMSSName),
		},
	}
	allSecretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.PortalServerSessionKeySecretName),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretName),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretName),
		},
		{
			ID: to.StringPtr(env.PortalServerSSHKeySecretName),
		},
	}
	partialSecretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: to.StringPtr(env.EncryptionSecretV2Name),
		},
		{
			ID: to.StringPtr(env.FrontendEncryptionSecretV2Name),
		},
	}
	instanceID := "testID"
	vms := []mgmtcompute.VirtualMachineScaleSetVM{
		{
			InstanceID: to.StringPtr(instanceID),
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}

	type mock func(*mock_keyvault.MockManager, *mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient)
	getSecretsFailed := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			allSecretItems, errors.New("generic error"),
		)
	}
	getSecretsSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			allSecretItems, nil,
		)
	}
	getNewSecretSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecret(ctx, gomock.Any()).Return(
			newSecretBundle, nil,
		)
	}
	getPartialSecretsSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().GetSecrets(ctx).Return(
			partialSecretItems, nil,
		)
	}
	setSecretSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}
	listVMSSFailed := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, gomock.Any()).Return(
			vmsss, errors.New("VM List Failed"),
		)
	}
	vmssListSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, gomock.Any()).Return(
			vmsss, nil,
		)
	}
	vmssVMsListSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().List(ctx, gomock.Any(), gomock.Any(), "", "", "").Return(
			vms, nil,
		)
	}
	restartSuccessful := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, gomock.Any(), gomock.Any(), instanceID, gomock.Any()).Return(nil)
	}
	healthyInstanceView := func(k *mock_keyvault.MockManager, vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), gomock.Any(), gomock.Any(), instanceID).Return(healthyVMSS, nil)
	}

	for _, tt := range []struct {
		name         string
		secretToFind string
		mocks        []mock
		wantErr      string
	}{
		{
			name: "return error if ensureAndRotateSecret fails",
			mocks: []mock{
				getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret passes without rotating any secret but ensureSecret fails",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret passes with rotating a missing secret but ensureSecret fails",
			mocks: []mock{
				getPartialSecretsSuccessful, getNewSecretSuccessful, getPartialSecretsSuccessful, getNewSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret, ensureSecret passes without rotating a secret but ensureSecretKey fails",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getSecretsSuccessful, getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name: "return error if ensureAndRotateSecret, ensureSecret passes with rotating a legacy secret but ensureSecretKey fails",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getSecretsFailed,
			},
			wantErr: "generic error",
		},
		{
			name: "return nil if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes without rotating a secret",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getSecretsSuccessful, getSecretsSuccessful,
			},
		},
		{
			name: "return error if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes with rotating secret in each ensure function call but restartoldscaleset failing",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getSecretsSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, listVMSSFailed,
			},
			wantErr: "VM List Failed",
		},
		{
			name: "return nil if ensureAndRotateSecret, ensureSecret, ensureSecretKey passes with rotating secret in each ensure function call and restartoldscaleset passess successfully",
			mocks: []mock{
				getSecretsSuccessful, getNewSecretSuccessful, getSecretsSuccessful, getNewSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getSecretsSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, getPartialSecretsSuccessful, setSecretSuccessful, vmssListSuccessful, vmssVMsListSuccessful, restartSuccessful, healthyInstanceView, vmssListSuccessful, vmssVMsListSuccessful, restartSuccessful, healthyInstanceView,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_keyvault.NewMockManager(controller)
			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			mockVMSSVM := mock_compute.NewMockVirtualMachineScaleSetVMsClient(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
				config: &RPConfig{
					RPResourceGroupName:      rgName,
					GatewayResourceGroupName: rgName,
				},
				serviceKeyvault: mockKV,
				portalKeyvault:  mockKV,
				vmss:            mockVMSS,
				vmssvms:         mockVMSSVM,
			}

			for _, m := range tt.mocks {
				m(mockKV, mockVMSS, mockVMSSVM)
			}

			err := d.configureServiceSecrets(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureAndRotateSecret(t *testing.T) {
	ctx := context.Background()
	secretExists := "secretExists"
	noSecretExists := "noSecretExists"
	secretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: &secretExists,
		},
	}
	nowUnixTime := date.NewUnixTimeFromSeconds(float64(time.Now().Unix()))
	oldUnixTime := date.NewUnixTimeFromSeconds(float64(time.Now().Add(-rotateSecretAfter).Unix()))
	newSecretBundle := azkeyvault.SecretBundle{
		Attributes: &azkeyvault.SecretAttributes{
			Created: &nowUnixTime,
		},
	}

	oldSecretBundle := azkeyvault.SecretBundle{
		Attributes: &azkeyvault.SecretAttributes{
			Created: &oldUnixTime,
		},
	}

	type mock func(*mock_keyvault.MockManager)
	getSecretsFailed := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, errors.New("generic error"),
		)
	}
	getSecretsSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, nil,
		)
	}
	getSecretFailed := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecret(ctx, secretExists).Return(
			newSecretBundle, errors.New("generic error"),
		)
	}
	getNewSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecret(ctx, secretExists).Return(
			newSecretBundle, nil,
		)
	}
	getOldSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecret(ctx, secretExists).Return(
			oldSecretBundle, nil,
		)
	}
	setSecretFails := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	setSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, gomock.Any(), gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name         string
		secretToFind string
		mocks        []mock
		wantErr      string
		wantBool     bool
	}{
		{
			name:         "return false and error if GetSecrets fails",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsFailed,
			},
			wantBool: false,
			wantErr:  "generic error",
		},
		{
			name:         "return false and error if GetSecrets passes but GetSecret fails for the found secret",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
				getSecretFailed,
			},
			wantBool: false,
			wantErr:  "generic error",
		},
		{
			name:         "return false and nil if GetSecrets and GetSecret passes and the secret is not too old",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
				getNewSecretSuccessful,
			},
			wantBool: false,
		},
		{
			name:         "return true and error if GetSecrets & GetSecret passes and the secret is old but new secret creation fails",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
				getOldSecretSuccessful,
				setSecretFails,
			},
			wantBool: true,
			wantErr:  "generic error",
		},
		{
			name:         "return true and nil if GetSecrets & GetSecret passes and the secret is old and new secret creation passes",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
				getOldSecretSuccessful,
				setSecretSuccessful,
			},
			wantBool: true,
		},
		{
			name:         "return true and nil if the secret is not present and new secret creation passes",
			secretToFind: noSecretExists,
			mocks: []mock{
				getSecretsSuccessful,
				setSecretSuccessful,
			},
			wantBool: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_keyvault.NewMockManager(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV)
			}

			got, err := d.ensureAndRotateSecret(ctx, mockKV, tt.secretToFind, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestEnsureSecret(t *testing.T) {
	ctx := context.Background()
	secretExists := "secretExists"
	noSecretExists := "noSecretExists"
	secretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: &secretExists,
		},
	}

	type mock func(*mock_keyvault.MockManager)
	getSecretsFailed := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, errors.New("generic error"),
		)
	}
	getSecretsSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, nil,
		)
	}
	setSecretFails := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	setSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name         string
		secretToFind string
		mocks        []mock
		wantErr      string
		wantBool     bool
	}{
		{
			name:         "return false and error if GetSecrets fails",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsFailed,
			},
			wantBool: false,
			wantErr:  "generic error",
		},
		{
			name:         "return false and nil if GetSecrets passes and secret is found",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
			},
			wantBool: false,
		},
		{
			name:         "return true and error if GetSecrets passes but secret is not found and new secret creation fails",
			secretToFind: noSecretExists,
			mocks: []mock{
				getSecretsSuccessful, setSecretFails,
			},
			wantBool: true,
			wantErr:  "generic error",
		},
		{
			name:         "return true and nil if GetSecrets passes but secret is not found and new secret creation also passes",
			secretToFind: noSecretExists,
			mocks: []mock{
				getSecretsSuccessful, setSecretSuccessful,
			},
			wantBool: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_keyvault.NewMockManager(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV)
			}

			got, err := d.ensureSecret(ctx, mockKV, tt.secretToFind, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestCreateSecret(t *testing.T) {
	ctx := context.Background()
	noSecretExists := "noSecretExists"

	type mock func(*mock_keyvault.MockManager)
	setSecretFails := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	setSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name           string
		secretToCreate string
		mocks          []mock
		wantErr        string
	}{
		{
			name:           "return error if new secret creation fails",
			secretToCreate: noSecretExists,
			mocks: []mock{
				setSecretFails,
			},
			wantErr: "generic error",
		},
		{
			name:           "return nil new secret creation passes",
			secretToCreate: noSecretExists,
			mocks: []mock{
				setSecretSuccessful,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_keyvault.NewMockManager(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV)
			}

			err := d.createSecret(ctx, mockKV, tt.secretToCreate, 8)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureSecretKey(t *testing.T) {
	ctx := context.Background()
	secretExists := "secretExists"
	noSecretExists := "noSecretExists"
	secretItems := []azkeyvault.SecretItem{
		{
			ID: to.StringPtr("test1"),
		},
		{
			ID: &secretExists,
		},
	}

	type mock func(*mock_keyvault.MockManager)
	getSecretsFailed := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, errors.New("generic error"),
		)
	}
	getSecretsSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().GetSecrets(ctx).Return(
			secretItems, nil,
		)
	}
	setSecretFails := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			errors.New("generic error"),
		)
	}
	setSecretSuccessful := func(k *mock_keyvault.MockManager) {
		k.EXPECT().SetSecret(ctx, noSecretExists, gomock.Any()).Return(
			nil,
		)
	}

	for _, tt := range []struct {
		name         string
		secretToFind string
		mocks        []mock
		wantErr      string
		wantBool     bool
	}{
		{
			name:         "return false and error if GetSecrets fails",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsFailed,
			},
			wantBool: false,
			wantErr:  "generic error",
		},
		{
			name:         "return false and nil if GetSecrets passes and secret is found",
			secretToFind: secretExists,
			mocks: []mock{
				getSecretsSuccessful,
			},
			wantBool: false,
		},
		{
			name:         "return true and error if GetSecrets passes but secret is not found and new secret creation fails",
			secretToFind: noSecretExists,
			mocks: []mock{
				getSecretsSuccessful, setSecretFails,
			},
			wantBool: true,
			wantErr:  "generic error",
		},
		{
			name:         "return true and nil if GetSecrets passes but secret is not found and new secret creation also passes",
			secretToFind: noSecretExists,
			mocks: []mock{
				getSecretsSuccessful, setSecretSuccessful,
			},
			wantBool: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_keyvault.NewMockManager(controller)

			d := deployer{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			for _, m := range tt.mocks {
				m(mockKV)
			}

			got, err := d.ensureSecretKey(ctx, mockKV, tt.secretToFind)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestRestartOldScalesets(t *testing.T) {
	ctx := context.Background()
	rgName := "testRG"
	rpVMSSName := rpVMSSPrefix + "test"
	invalidVMSSName := "other-vmss"
	invalidVMSSs := []mgmtcompute.VirtualMachineScaleSet{
		{
			Name: to.StringPtr(invalidVMSSName),
		},
	}
	vmsss := []mgmtcompute.VirtualMachineScaleSet{
		{
			Name: to.StringPtr(rpVMSSName),
		},
	}
	instanceID := "testID"
	vms := []mgmtcompute.VirtualMachineScaleSetVM{
		{
			InstanceID: to.StringPtr(instanceID),
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}

	type mock func(*mock_compute.MockVirtualMachineScaleSetsClient, *mock_compute.MockVirtualMachineScaleSetVMsClient)
	listVMSSFailed := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, rgName).Return(
			vmsss, errors.New("generic error"),
		)
	}
	invalidVMSSSList := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, rgName).Return(
			invalidVMSSs, nil,
		)
	}
	vmssListSuccessful := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmss.EXPECT().List(ctx, rgName).Return(
			vmsss, nil,
		)
	}
	vmssVMsListFailed := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().List(ctx, rgName, rpVMSSName, "", "", "").Return(
			vms, errors.New("generic error"),
		)
	}
	vmssVMsListSuccessful := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().List(ctx, rgName, rpVMSSName, "", "", "").Return(
			vms, nil,
		)
	}
	restartSuccessful := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().RunCommandAndWait(ctx, rgName, rpVMSSName, instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{rpRestartScript},
		}).Return(nil)
	}
	healthyInstanceView := func(vmss *mock_compute.MockVirtualMachineScaleSetsClient, vmssvms *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		vmssvms.EXPECT().GetInstanceView(gomock.Any(), rgName, rpVMSSName, instanceID).Return(healthyVMSS, nil)
	}

	for _, tt := range []struct {
		name              string
		resourceGroupName string
		mocks             []mock
		wantErr           string
	}{
		{
			name:              "Don't continue if vmss list fails",
			resourceGroupName: rgName,
			mocks:             []mock{listVMSSFailed},
			wantErr:           "generic error",
		},
		{
			name:              "Don't continue if vmss list has an invalid vmss name",
			resourceGroupName: rgName,
			mocks:             []mock{invalidVMSSSList},
			wantErr:           "400: InvalidResource: : provided vmss other-vmss does not match RP or gateway prefix",
		},
		{
			name:              "Don't continue if vmssvms list fails",
			resourceGroupName: rgName,
			mocks:             []mock{vmssListSuccessful, vmssVMsListFailed},
			wantErr:           "generic error",
		},
		{
			name:              "Restart is successful for the VMs in VMSS",
			resourceGroupName: rgName,
			mocks:             []mock{vmssListSuccessful, vmssVMsListSuccessful, restartSuccessful, healthyInstanceView},
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
			}

			for _, m := range tt.mocks {
				m(mockVMSS, mockVMSSVM)
			}

			err := d.restartOldScalesets(ctx, tt.resourceGroupName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestRestartOldScaleset(t *testing.T) {
	ctx := context.Background()
	otherVMSSName := "other-vmss"
	rgName := "testRG"
	gwyVMSSName := gatewayVMSSPrefix + "test"
	rpVMSSName := rpVMSSPrefix + "test"
	instanceID := "testID"
	vms := []mgmtcompute.VirtualMachineScaleSetVM{
		{
			InstanceID: to.StringPtr(instanceID),
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}

	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient)
	listVMSSFailed := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().List(ctx, rgName, gwyVMSSName, "", "", "").Return(
			vms, errors.New("generic error"),
		)
	}
	listVMSSSuccessful := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().List(ctx, rgName, gomock.Any(), "", "", "").Return(
			vms, nil,
		)
	}
	gatewayRestartFailed := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().RunCommandAndWait(ctx, rgName, gwyVMSSName, instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{gatewayRestartScript},
		}).Return(
			errors.New("generic error"),
		)
	}
	rpRestartFailed := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().RunCommandAndWait(ctx, rgName, rpVMSSName, instanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{rpRestartScript},
		}).Return(
			errors.New("generic error"),
		)
	}
	restartSuccessful := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().RunCommandAndWait(ctx, rgName, gomock.Any(), instanceID, gomock.Any()).Return(nil)
	}
	healthyInstanceView := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(gomock.Any(), rgName, gomock.Any(), instanceID).Return(healthyVMSS, nil)
	}
	for _, tt := range []struct {
		name              string
		vmssName          string
		resourceGroupName string
		mocks             []mock
		wantErr           string
	}{
		{
			name:     "Return an error if the VMSS is not gateway or RP",
			vmssName: otherVMSSName,
			wantErr:  "400: InvalidResource: : provided vmss other-vmss does not match RP or gateway prefix",
		},
		{
			name:              "list VMSS failed",
			vmssName:          gwyVMSSName,
			resourceGroupName: rgName,
			mocks:             []mock{listVMSSFailed},
			wantErr:           "generic error",
		},
		{
			name:              "gateway restart script failed",
			vmssName:          gwyVMSSName,
			resourceGroupName: rgName,
			mocks:             []mock{listVMSSSuccessful, gatewayRestartFailed},
			wantErr:           "generic error",
		},
		{
			name:              "rp restart script failed",
			vmssName:          rpVMSSName,
			resourceGroupName: rgName,
			mocks:             []mock{listVMSSSuccessful, rpRestartFailed},
			wantErr:           "generic error",
		},
		{
			name:              "restart script passes and wait for readiness is successful",
			vmssName:          rpVMSSName,
			resourceGroupName: rgName,
			mocks:             []mock{listVMSSSuccessful, restartSuccessful, healthyInstanceView},
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
				m(mockVMSS)
			}

			err := d.restartOldScaleset(ctx, tt.vmssName, tt.resourceGroupName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestWaitForReadiness(t *testing.T) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	vmmssName := "testVMSS"
	vmInstanceID := "testVMInstanceID"
	testRG := "testRG"
	unhealthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/unhealthy"),
			},
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}
	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient)
	unhealthyInstanceView := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctxTimeout, testRG, vmmssName, vmInstanceID).Return(unhealthyVMSS, nil).AnyTimes()
	}
	healthyInstanceView := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctxTimeout, testRG, vmmssName, vmInstanceID).Return(healthyVMSS, nil)
	}
	for _, tt := range []struct {
		name              string
		ctx               context.Context
		cancel            context.CancelFunc
		vmssName          string
		vmInstanceID      string
		resourceGroupName string
		mocks             []mock
		wantErr           string
	}{
		{
			name:              "fail after context times out",
			ctx:               ctxTimeout,
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: testRG,
			mocks: []mock{
				unhealthyInstanceView,
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name:              "run successfully after confirming healthy status",
			ctx:               ctxTimeout,
			cancel:            cancel,
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: testRG,
			mocks: []mock{
				healthyInstanceView,
			},
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
				m(mockVMSS)
			}

			defer cancel()
			err := d.waitForReadiness(tt.ctx, tt.resourceGroupName, tt.vmssName, tt.vmInstanceID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestIsVMInstanceHealthy(t *testing.T) {
	ctx := context.Background()
	vmmssName := "testVMSS"
	vmInstanceID := "testVMInstanceID"
	rpRGName := "testRPRG"
	gatewayRGName := "testGatewayRG"
	unhealthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/unhealthy"),
			},
		},
	}
	healthyVMSS := mgmtcompute.VirtualMachineScaleSetVMInstanceView{
		VMHealth: &mgmtcompute.VirtualMachineHealthStatus{
			Status: &mgmtcompute.InstanceViewStatus{
				Code: to.StringPtr("HealthState/healthy"),
			},
		},
	}

	type mock func(*mock_compute.MockVirtualMachineScaleSetVMsClient)
	getRPInstanceViewFailed := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctx, rpRGName, vmmssName, vmInstanceID).Return(
			unhealthyVMSS, errors.New("generic error"),
		)
	}
	getGatewayInstanceViewFailed := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctx, gatewayRGName, vmmssName, vmInstanceID).Return(
			unhealthyVMSS, errors.New("generic error"),
		)
	}
	unhealthyInstanceView := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctx, gatewayRGName, vmmssName, vmInstanceID).Return(unhealthyVMSS, nil)
	}
	healthyInstanceView := func(c *mock_compute.MockVirtualMachineScaleSetVMsClient) {
		c.EXPECT().GetInstanceView(ctx, gatewayRGName, vmmssName, vmInstanceID).Return(healthyVMSS, nil)
	}
	for _, tt := range []struct {
		name              string
		vmssName          string
		vmInstanceID      string
		resourceGroupName string
		mocks             []mock
		wantBool          bool
	}{
		{
			name:              "return false if GetInstanceView failed for RP resource group",
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: rpRGName,
			mocks: []mock{
				getRPInstanceViewFailed,
			},
			wantBool: false,
		},
		{
			name:              "return false if GetInstanceView failed for Gateway resource group",
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: gatewayRGName,
			mocks: []mock{
				getGatewayInstanceViewFailed,
			},
			wantBool: false,
		},
		{
			name:              "return false if GetInstanceView return unhealthy VM",
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: gatewayRGName,
			mocks: []mock{
				unhealthyInstanceView,
			},
			wantBool: false,
		},
		{
			name:              "return true if GetInstanceView return healthy VM",
			vmssName:          vmmssName,
			vmInstanceID:      vmInstanceID,
			resourceGroupName: gatewayRGName,
			mocks: []mock{
				healthyInstanceView,
			},
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
				m(mockVMSS)
			}

			got := d.isVMInstanceHealthy(ctx, tt.resourceGroupName, tt.vmssName, tt.vmInstanceID)
			if tt.wantBool != got {
				t.Errorf("%#v", got)
			}
		})
	}
}
