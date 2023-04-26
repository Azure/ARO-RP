package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
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
	genericDeploymentFailed := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, *deployment).Return(
			&azure.ServiceError{
				Code: "DeploymentFailed",
				Details: []map[string]interface{}{
					{
						"code":    "BadRequest",
						"message": "{\r\n  \"code\": \"FooErrorCode\",\r\n  \"message\": \"Not something we can deal with automatically.\"\r\n}",
					},
				},
			},
		)
	}
	deploymentFailedLBNotFound := func(d *mock_features.MockDeploymentsClient, v *mock_vmsscleaner.MockInterface) {
		d.EXPECT().CreateOrUpdateAndWait(ctx, rgName, deploymentName, *deployment).Return(
			&azure.ServiceError{
				Code: "DeploymentFailed",
				Details: []map[string]interface{}{
					{
						"code":    "BadRequest",
						"message": fmt.Sprintf("{\r\n  \"code\": \"ResourceNotFound\",\r\n  \"message\": \"The Resource 'Microsoft.Network/loadBalancers/rp-lb' under resource group '%s' was not found. For more details please go to https://aka.ms/ARMResourceNotFoundFix Activity ID: 00000000-0000-0000-0000-000000000000.\"\r\n}", rgName),
					},
				},
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
			name:  "continue after initial deploymentFailed with LB not found error; don't continue if shouldNotRetry",
			clean: true,
			mocks: []mock{
				deploymentFailedLBNotFound, otherDeploymentError, shouldNotRetry,
			},
			wantErr: `Code="Computer says 'no'" Message=""`,
		},
		{
			name:  "don't continue if genericDeploymentFailed and shouldNotRetry",
			clean: true,
			mocks: []mock{
				genericDeploymentFailed, shouldNotRetry,
			},
			wantErr: "Code=\"DeploymentFailed\" Message=\"\" Details=[{\"code\":\"BadRequest\",\"message\":\"{\\r\\n  \\\"code\\\": \\\"FooErrorCode\\\",\\r\\n  \\\"message\\\": \\\"Not something we can deal with automatically.\\\"\\r\\n}\"}]",
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

func TestCheckForKnownError(t *testing.T) {
	rgName := "testRG"

	rpLBNotFound := &azure.ServiceError{
		Code: "DeploymentFailed",
		Details: []map[string]interface{}{
			{
				"code":    "BadRequest",
				"message": fmt.Sprintf("{\r\n  \"code\": \"ResourceNotFound\",\r\n  \"message\": \"The Resource 'Microsoft.Network/loadBalancers/rp-lb' under resource group '%s' was not found. For more details please go to https://aka.ms/ARMResourceNotFoundFix Activity ID: 00000000-0000-0000-0000-000000000000.\"\r\n}", rgName),
			},
		},
	}

	unfamiliarError := &azure.ServiceError{
		Code: "DeploymentFailed",
		Details: []map[string]interface{}{
			{
				"code":    "BadRequest",
				"message": "{\r\n  \"code\": \"Unfamiliar\",\r\n  \"message\": \"This is an unfamiliar error.\"\r\n}",
			},
		},
	}

	multipleErrors := &azure.ServiceError{
		Code: "DeploymentFailed",
		Details: []map[string]interface{}{
			{
				"code":    "BadRequest",
				"message": "{\r\n  \"code\": \"Unfamiliar\",\r\n  \"message\": \"This is an unfamiliar error.\"\r\n}",
			},
			{
				"code":    "BadRequest",
				"message": fmt.Sprintf("{\r\n  \"code\": \"ResourceNotFound\",\r\n  \"message\": \"The Resource 'Microsoft.Network/loadBalancers/rp-lb' under resource group '%s' was not found. For more details please go to https://aka.ms/ARMResourceNotFoundFix Activity ID: 00000000-0000-0000-0000-000000000000.\"\r\n}", rgName),
			},
		},
	}

	multipleErrorsFirstKnown := &azure.ServiceError{
		Code: "DeploymentFailed",
		Details: []map[string]interface{}{
			{
				"code":    "BadRequest",
				"message": fmt.Sprintf("{\r\n  \"code\": \"ResourceNotFound\",\r\n  \"message\": \"The Resource 'Microsoft.Network/loadBalancers/rp-lb' under resource group '%s' was not found. For more details please go to https://aka.ms/ARMResourceNotFoundFix Activity ID: 00000000-0000-0000-0000-000000000000.\"\r\n}", rgName),
			},
			{
				"code":    "BadRequest",
				"message": "{\r\n  \"code\": \"Unfamiliar\",\r\n  \"message\": \"This is an unfamiliar error.\"\r\n}",
			},
		},
	}

	nestedErrorDoesntUnmarshal := &azure.ServiceError{
		Code: "DeploymentFailed",
		Details: []map[string]interface{}{
			{
				"code":    "BadRequest",
				"message": "I am just a regular string and not a JSON-encoded string representing another ServiceError.",
			},
		},
	}

	for _, tt := range []struct {
		name          string
		serviceError  *azure.ServiceError
		deployAttempt int
		want          KnownDeploymentErrorType
		wantErr       string
	}{
		{
			name:          "Known RP LB ResourceNotFound error",
			serviceError:  rpLBNotFound,
			deployAttempt: 0,
			want:          KnownDeploymentErrorTypeRPLBNotFound,
		},
		{
			name:          "Known RP LB ResourceNotFound, but not first deploy attempt",
			serviceError:  rpLBNotFound,
			deployAttempt: 1,
			want:          "",
		},
		{
			name:          "Random unfamiliar error",
			serviceError:  unfamiliarError,
			deployAttempt: 0,
			want:          "",
		},
		{
			name:          "Multiple nested errors, first one is not familiar",
			serviceError:  multipleErrors,
			deployAttempt: 0,
			want:          "",
		},
		{
			name:          "Multiple nested errors, first one is familiar",
			serviceError:  multipleErrorsFirstKnown,
			deployAttempt: 0,
			want:          KnownDeploymentErrorTypeRPLBNotFound,
		},
		{
			name:          "No innermost nested error",
			serviceError:  nestedErrorDoesntUnmarshal,
			deployAttempt: 0,
			want:          "",
			wantErr:       "invalid character 'I' looking for beginning of value",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			d := deployer{}

			got, err := d.checkForKnownError(tt.serviceError, tt.deployAttempt)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.want != got {
				t.Errorf("%#v", got)
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
