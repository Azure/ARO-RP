package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-01-15/documentdb"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_containerservice "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/containerservice"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_msi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/msi"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_vmsscleaner "github.com/Azure/ARO-RP/pkg/util/mocks/vmsscleaner"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestDeploy(t *testing.T) {
	ctx := context.Background()
	rgName := "testRG"
	deploymentName := "testDeployment"
	vmssName := "testVMSS"

	deployment := &mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   nil,
			Mode:       mgmtfeatures.Incremental,
			Parameters: nil,
		},
	}

	type mock func(*mock_features.MockDeploymentsClient, *mock_vmsscleaner.MockInterface)

	deploymentFailed := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, *deployment).Return(
			&azure.ServiceError{
				Code: "DeploymentFailed",
			},
		)
	}
	otherDeploymentError := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, *deployment).Return(
			&azure.ServiceError{
				Code: "Computer says 'no'",
			},
		)
	}
	deploymentSuccessful := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, *deployment).Return(nil)
	}
	shouldNotRetry := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		v.EXPECT().RemoveFailedNewScaleset(ctx, rgName, vmssName).Return(false)
	}
	shouldRetry := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		v.EXPECT().RemoveFailedNewScaleset(ctx, rgName, vmssName).Return(true)
	}

	for _, tt := range []struct {
		name    string
		clean   bool
		mocks   []mock
		wantErr string
	}{
		{
			name: "otherDeploymentError, VMSS cleanup not enabled",
			mocks: []mock{
				otherDeploymentError,
			},
			wantErr: `Code="Computer says 'no'" Message=""`,
		},
		{
			name:  "continue after initial deploymentFailed; don't continue if shouldNotRetry",
			clean: true,
			mocks: []mock{
				deploymentFailed, otherDeploymentError, shouldNotRetry,
			},
			wantErr: `Code="Computer says 'no'" Message=""`,
		},
		{
			name:  "continue if shouldRetry after error",
			clean: true,
			mocks: []mock{
				otherDeploymentError, shouldRetry, otherDeploymentError, shouldRetry, otherDeploymentError, shouldRetry,
			},
			wantErr: `Code="Computer says 'no'" Message=""`,
		},
		{
			name:  "otherDeploymentError, shouldRetry; deploymentSuccessful",
			clean: true,
			mocks: []mock{
				otherDeploymentError, shouldRetry, deploymentSuccessful,
			},
		},
		{
			name:  "deploymentSuccessful",
			clean: true,
			mocks: []mock{
				deploymentSuccessful,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockDeployments := mock_features.NewMockDeploymentsClient(controller)
			mockVMSSCleaner := mock_vmsscleaner.NewMockInterface(controller)

			d := deployer{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				deployments: mockDeployments,
				vmssCleaner: mockVMSSCleaner,
				config: &RPConfig{
					Configuration: &Configuration{
						VMSSCleanupEnabled: &tt.clean,
					},
				},
			}

			for _, m := range tt.mocks {
				m(mockDeployments, mockVMSSCleaner)
			}

			err := d.deploy(ctx, rgName, deploymentName, vmssName, *deployment)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}
		})
	}
}
func TestGetParameters(t *testing.T) {
	databaseAccountName := to.StringPtr("databaseAccountName")
	adminApiCaBundle := to.StringPtr("adminApiCaBundle")
	extraClusterKeyVaultAccessPolicies := []interface{}{"a", "b", 1}
	for _, tt := range []struct {
		name   string
		ps     map[string]interface{}
		config Configuration
		want   arm.Parameters
	}{
		{
			name: "when no parameters are present only default is returned",
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
		{
			name: "when all parameters present, everything is copied",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName:                databaseAccountName,
				AdminAPICABundle:                   adminApiCaBundle,
				ExtraClusterKeyvaultAccessPolicies: extraClusterKeyVaultAccessPolicies,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
					"extraClusterKeyvaultAccessPolicies": {
						Value: extraClusterKeyVaultAccessPolicies,
					},
					"adminApiCaBundle": {
						Value: adminApiCaBundle,
					},
				},
			},
		},
		{
			name: "when parameters with nil config are present, they are not returned",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName: databaseAccountName,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
				},
			},
		},
		{
			name: "when nil slice parameter is present it is skipped",
			ps: map[string]interface{}{
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
		{
			name: "when malformed parameter is present, it is skipped",
			ps: map[string]interface{}{
				"dutabaseAccountName": nil,
			},
			config: Configuration{
				DatabaseAccountName: databaseAccountName,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := deployer{
				config: &RPConfig{Configuration: &tt.config},
			}

			got := d.getParameters(tt.ps)

			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("%#v", got)
			}
		})
	}
}

func TestRPParameters(t *testing.T) {
	adminApiCaBundle := to.StringPtr("adminApiCaBundle")
	RPImagePrefix := to.StringPtr("RPImagePrefix")
	hiveKubeConfig := to.ByteSlicePtr([]byte("alreadybase64"))

	principalID := uuid.MustFromString(uuid.DefaultGenerator.Generate())

	for _, tt := range []struct {
		name        string
		mocks       func(*mock_containerservice.MockManagedClustersClient)
		config      Configuration
		want        arm.Parameters
		wantEntries []map[string]types.GomegaMatcher
	}{
		{
			name: "hive kubeconfig fetched",
			mocks: func(mcc *mock_containerservice.MockManagedClustersClient) {
				mcc.EXPECT().ListClusterUserCredentials(gomock.Any(), "rp-eastus", "aro-aks-cluster-001", "").Return(mgmtcontainerservice.CredentialResults{
					Kubeconfigs: &[]mgmtcontainerservice.CredentialResult{
						{
							Name:  to.StringPtr("example"),
							Value: hiveKubeConfig,
						},
					},
				}, nil)
			},
			config: Configuration{
				AdminAPICABundle: adminApiCaBundle,
				RPImagePrefix:    RPImagePrefix,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"adminApiCaBundle": {
						Value: base64.StdEncoding.EncodeToString([]byte(*adminApiCaBundle)),
					},
					"azureCloudName": {
						Value: "AzureCloud",
					},
					"gatewayServicePrincipalId": {
						Value: principalID.String(),
					},
					"rpServicePrincipalId": {
						Value: principalID.String(),
					},
					"rpImage": {
						Value: "RPImagePrefix:ver1234",
					},
					"gatewayResourceGroupName": {
						Value: "gwy",
					},
					"keyvaultDNSSuffix": {
						Value: "vault.azure.net",
					},
					"hiveKubeconfig": {
						Value: "alreadybase64",
					},
					"vmssName": {
						Value: "ver1234",
					},
					"ipRules": {
						Value: []mgmtdocumentdb.IPAddressOrRange{},
					},
				},
			},
		},
		{
			name: "hive kubeconfig missing",
			mocks: func(mcc *mock_containerservice.MockManagedClustersClient) {
				mcc.EXPECT().ListClusterUserCredentials(gomock.Any(), "rp-eastus", "aro-aks-cluster-001", "").Return(mgmtcontainerservice.CredentialResults{}, errors.New("whoops"))
			},
			config: Configuration{
				AdminAPICABundle: adminApiCaBundle,
				RPImagePrefix:    RPImagePrefix,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"adminApiCaBundle": {
						Value: base64.StdEncoding.EncodeToString([]byte(*adminApiCaBundle)),
					},
					"azureCloudName": {
						Value: "AzureCloud",
					},
					"gatewayServicePrincipalId": {
						Value: principalID.String(),
					},
					"rpServicePrincipalId": {
						Value: principalID.String(),
					},
					"rpImage": {
						Value: "RPImagePrefix:ver1234",
					},
					"gatewayResourceGroupName": {
						Value: "gwy",
					},
					"keyvaultDNSSuffix": {
						Value: "vault.azure.net",
					},
					"vmssName": {
						Value: "ver1234",
					},
					"ipRules": {
						Value: []mgmtdocumentdb.IPAddressOrRange{},
					},
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`whoops`),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			hook, entry := testlog.New()

			mockUserAssignedIdentities := mock_msi.NewMockUserAssignedIdentitiesClient(controller)
			mcc := mock_containerservice.NewMockManagedClustersClient(controller)
			mockenv := mock_env.NewMockCore(controller)

			mockUserAssignedIdentities.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(mgmtmsi.Identity{
				UserAssignedIdentityProperties: &mgmtmsi.UserAssignedIdentityProperties{
					PrincipalID: &principalID,
				},
			}, nil)
			mockenv.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			tt.mocks(mcc)

			d := deployer{
				log:     entry,
				env:     mockenv,
				version: "ver1234",
				config: &RPConfig{
					Location:                 "eastus",
					Configuration:            &tt.config,
					GatewayResourceGroupName: "gwy",
					RPResourceGroupName:      "rp",
				},
				userassignedidentities: mockUserAssignedIdentities,
				managedclusters:        mcc,
			}

			_, params, err := d.rpTemplateAndParameters(ctx)
			if err != nil {
				t.Fatal(err)
			}

			for _, i := range deep.Equal(params, &tt.want) {
				t.Error(i)
			}

			err = testlog.AssertLoggingOutput(hook, tt.wantEntries)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
