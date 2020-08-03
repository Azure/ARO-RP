package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
	test_log "github.com/Azure/ARO-RP/test/util/log"
)

func failingFunc(context.Context) error { return errors.New("oh no!") }

var clusterOperators = &configv1.ClusterOperator{
	ObjectMeta: metav1.ObjectMeta{
		Name: "console",
	},
	Status: configv1.ClusterOperatorStatus{
		Versions: []configv1.OperandVersion{
			{
				Name:    "operator",
				Version: "4.3.0",
			},
			{
				Name:    "operator-good",
				Version: "4.3.1",
			},
		},
	},
}

var clusterVersion = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		Desired: configv1.Update{
			Version: "1.2.3",
		},
	},
}

func TestStepRunnerWithInstaller(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		steps       []steps.Step
		wantEntries []test_log.ExpectedLogEntry
		wantErr     string
		configcli   *fake.Clientset
	}{
		{
			name: "Failed step run will log cluster version and cluster operator information if available",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []test_log.ExpectedLogEntry{
				{
					Level:   logrus.InfoLevel,
					Message: `running step [Action github.com/Azure/ARO-RP/pkg/install.failingFunc]`,
				},
				{
					Level:   logrus.ErrorLevel,
					Message: "step [Action github.com/Azure/ARO-RP/pkg/install.failingFunc] encountered error: oh no!",
				},
				{
					Level:        logrus.InfoLevel,
					MessageRegex: `github.com/Azure/ARO-RP/pkg/install.\(\*Installer\).logClusterVersion\-fm: {.*"version":"1.2.3".*}`,
				},
				{
					Level:        logrus.InfoLevel,
					MessageRegex: `github.com/Azure/ARO-RP/pkg/install.\(\*Installer\).logClusterOperators\-fm: {.*"versions":\[{"name":"operator","version":"4.3.0"},{"name":"operator\-good","version":"4.3.1"}\].*}`,
				},
			},
			configcli: fake.NewSimpleClientset(clusterVersion, clusterOperators),
		},
		{
			name: "Failed step run will not crash if it cannot get the clusterversions or clusteroperators",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []test_log.ExpectedLogEntry{
				{
					Level:   logrus.InfoLevel,
					Message: `running step [Action github.com/Azure/ARO-RP/pkg/install.failingFunc]`,
				},
				{
					Level:   logrus.ErrorLevel,
					Message: "step [Action github.com/Azure/ARO-RP/pkg/install.failingFunc] encountered error: oh no!",
				},
				{
					Level:   logrus.ErrorLevel,
					Message: `clusterversions.config.openshift.io "version" not found`,
				},
				{
					Level:   logrus.InfoLevel,
					Message: `github.com/Azure/ARO-RP/pkg/install.(*Installer).logClusterOperators-fm: {"metadata":{},"items":null}`,
				},
			},
			configcli: fake.NewSimpleClientset(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := test_log.NewCapturingLogger()
			i := &Installer{
				log:       log,
				configcli: tt.configcli,
			}

			err := i.runSteps(ctx, tt.steps)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			for _, e := range test_log.AssertLoggingOutput(h, tt.wantEntries) {
				t.Error(e)
			}
		})
	}
}

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
	databaseDoc, err := json.Marshal(clusterdoc)
	if err != nil {
		t.Error(err)
		return
	}

	openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
	openshiftClusters.EXPECT().
		PatchWithLease(gomock.Any(), clusterdoc.Key, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, f func(doc *api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
			// Load what the database would have right now
			docFromDatabase := &api.OpenShiftClusterDocument{}
			err := json.Unmarshal(databaseDoc, &docFromDatabase)
			if err != nil {
				t.Error(err)
				return nil, err
			}

			err = f(docFromDatabase)
			if err != nil {
				t.Error("PatchWithLease failed")
				return nil, err
			}

			// Save what would be stored in the db
			databaseDoc, err = json.Marshal(docFromDatabase)
			if err != nil {
				t.Error(err)
				return nil, err
			}

			return docFromDatabase, err
		})

	i := &Installer{
		doc: clusterdoc,
		db:  openshiftClusters,
	}
	err = i.addResourceProviderVersion(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	// Check it was set to the correct value in the database
	updatedClusterDoc := &api.OpenShiftClusterDocument{}
	err = json.Unmarshal(databaseDoc, &updatedClusterDoc)
	if err != nil {
		t.Error(err)
		return
	}
	if updatedClusterDoc.OpenShiftCluster.Properties.ProvisionedBy != version.GitCommit {
		t.Error("version was not added")
	}
}
