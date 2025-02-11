package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const deploymentName = "test"

func TestDeployARMTemplate(t *testing.T) {
	ctx := context.Background()

	resourceGroup := "fakeResourceGroup"

	armTemplate := &Template{}
	params := map[string]interface{}{}

	deployment := mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   armTemplate,
			Parameters: params,
			Mode:       mgmtfeatures.Incremental,
		},
	}

	activeErr := autorest.NewErrorWithError(azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "DeploymentActive"},
	}, "", "", nil, "")

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_features.MockDeploymentsClient)
		wantErr string
	}{
		{
			name: "Deployment successful with no errors",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(nil)
			},
		},
		{
			name: "Deployment active error, then wait successfully",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(activeErr)
				dc.EXPECT().
					Wait(ctx, resourceGroup, deploymentName).
					Return(nil)
			},
		},
		{
			name: "Deployment active error, then timeout",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(activeErr)
				dc.EXPECT().
					Wait(ctx, resourceGroup, deploymentName).
					Return(wait.ErrorInterrupted(errors.New("timed out waiting for the condition")))
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name: "DetailedError which should be returned to user",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(autorest.DetailedError{
						Original: &azure.ServiceError{
							Code: "AccountIsDisabled",
						},
					})
			},
			wantErr: `400: DeploymentFailed: : Deployment failed. Details: : : {"code":"AccountIsDisabled","message":"","target":null,"details":null,"innererror":null,"additionalInfo":null}`,
		},
		{
			name: "ServiceError which should be returned to user",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(&azure.ServiceError{
						Code: "AccountIsDisabled",
					})
			},
			wantErr: `400: DeploymentFailed: : Deployment failed. Details: : : {"code":"AccountIsDisabled","message":"","target":null,"details":null,"innererror":null,"additionalInfo":null}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			deploymentsClient := mock_features.NewMockDeploymentsClient(controller)
			tt.mocks(deploymentsClient)

			log := logrus.NewEntry(logrus.StandardLogger())

			err := DeployTemplate(ctx, log, deploymentsClient, resourceGroup, deploymentName, armTemplate, params)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
