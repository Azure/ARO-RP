package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_vmsscleaner "github.com/Azure/ARO-RP/pkg/util/mocks/vmsscleaner"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
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
