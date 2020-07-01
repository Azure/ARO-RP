package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestDeployARMTemplate(t *testing.T) {
	ctx := context.Background()

	resourceGroup := "fakeResourceGroup"

	armTemplate := &arm.Template{}
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
					Return(wait.ErrWaitTimeout)
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

			i := &Installer{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				deployments: deploymentsClient,
			}

			err := i.deployARMTemplate(ctx, resourceGroup, "test", armTemplate, params)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestAddResourceProviderVersion(t *testing.T) {
	ctx := context.Background()

	clusterdoc := &api.OpenShiftClusterDocument{
		Key: "test",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{},
		},
	}
	controller := gomock.NewController(t)
	defer controller.Finish()

	// The original, as-in-database version of clusterdoc
	databaseDoc, _ := json.Marshal(clusterdoc)

	openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
	openshiftClusters.EXPECT().
		PatchWithLease(gomock.Any(), clusterdoc.Key, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, f func(doc *api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {

			// Load what the database would have right now
			docFromDatabase := &api.OpenShiftClusterDocument{}
			json.Unmarshal(databaseDoc, &docFromDatabase)

			err := f(docFromDatabase)

			// Save what would be stored in the db
			databaseDoc, _ = json.Marshal(docFromDatabase)
			return docFromDatabase, err
		})

	i := &Installer{
		doc: clusterdoc,
		db:  openshiftClusters,
	}
	err := i.addResourceProviderVersion(ctx)

	if err != nil {
		t.Error(err)
	}

	// Check it was set to the correct value in the database
	updatedClusterDoc := &api.OpenShiftClusterDocument{}
	json.Unmarshal(databaseDoc, &updatedClusterDoc)
	if updatedClusterDoc.OpenShiftCluster.Properties.ProvisionedBy != version.GitCommit {
		t.Error("version was not added")
	}
}
