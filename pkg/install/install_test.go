package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_resources "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/resources"
)

func TestDeployARMTemplate(t *testing.T) {
	ctx := context.Background()

	resourceGroup := "fakeResourceGroup"

	armTemplate := &arm.Template{}
	params := map[string]interface{}{}

	deployment := mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   armTemplate,
			Parameters: params,
			Mode:       mgmtresources.Incremental,
		},
	}

	activeErr := autorest.NewErrorWithError(azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "DeploymentActive"},
	}, "", "", nil, "")

	fakeQuotaErrMsg := "Quota exceeded"
	quotaErr := autorest.NewErrorWithError(&azure.ServiceError{
		Details: []map[string]interface{}{{
			"code":    "QuotaExceeded",
			"message": fakeQuotaErrMsg,
		}}}, "", "", nil, "")

	cloudQuotaErr := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeQuotaExceeded, fakeQuotaErrMsg, "")

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_resources.MockDeploymentsClient)
		wantErr error
	}{
		{
			name: "Deployment successful with no errors",
			mocks: func(dc *mock_resources.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(nil)
			},
		},
		{
			name: "Deployment active error, then wait successfully",
			mocks: func(dc *mock_resources.MockDeploymentsClient) {
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
			mocks: func(dc *mock_resources.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(activeErr)
				dc.EXPECT().
					Wait(ctx, resourceGroup, deploymentName).
					Return(wait.ErrWaitTimeout)
			},
			wantErr: wait.ErrWaitTimeout,
		},
		{
			name: "Resource quota exceeded error",
			mocks: func(dc *mock_resources.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(quotaErr)
			},
			wantErr: cloudQuotaErr,
		},
		{
			name: "Deployment active error, then resource quota exceeded error",
			mocks: func(dc *mock_resources.MockDeploymentsClient) {
				dc.EXPECT().
					CreateOrUpdateAndWait(ctx, resourceGroup, deploymentName, deployment).
					Return(activeErr)
				dc.EXPECT().
					Wait(ctx, resourceGroup, deploymentName).
					Return(quotaErr)
			},
			wantErr: cloudQuotaErr,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			deploymentsClient := mock_resources.NewMockDeploymentsClient(controller)
			tt.mocks(deploymentsClient)

			i := &Installer{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				deployments: deploymentsClient,
			}

			err := i.deployARMTemplate(ctx, resourceGroup, "test", armTemplate, params)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Error(err)
			}
		})
	}
}
