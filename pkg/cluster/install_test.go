package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	fakeconfig "github.com/openshift/client-go/config/clientset/versioned/fake"
	fakeoperator "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func failingFunc(context.Context) error { return errors.New("oh no!") }

var clusterOperator = &configv1.ClusterOperator{
	ObjectMeta: metav1.ObjectMeta{
		Name: "operator",
	},
}

var clusterVersion = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
}

var node = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "node",
	},
}

var ingressController = &operatorv1.IngressController{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "openshift-ingress-operator",
		Name:      "ingress-controller",
	},
}

func TestStepRunnerWithInstaller(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name          string
		steps         []steps.Step
		wantEntries   []map[string]types.GomegaMatcher
		wantErr       string
		kubernetescli *fake.Clientset
		configcli     *fakeconfig.Clientset
		operatorcli   *fakeoperator.Clientset
	}{
		{
			name: "Failed step run will log cluster version, cluster operator status, and ingress information if available",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`running step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc]`),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc] encountered error: oh no!"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)github.com/Azure/ARO-RP/pkg/cluster.\(\*manager\).logClusterVersion\-fm:.*"name": "version"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)github.com/Azure/ARO-RP/pkg/cluster.\(\*manager\).logNodes\-fm:.*"name": "node"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)github.com/Azure/ARO-RP/pkg/cluster.\(\*manager\).logClusterOperators\-fm:.*"name": "operator"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)github.com/Azure/ARO-RP/pkg/cluster.\(\*manager\).logIngressControllers\-fm:.*"name": "ingress-controller"`),
				},
			},
			kubernetescli: fake.NewSimpleClientset(node),
			configcli:     fakeconfig.NewSimpleClientset(clusterVersion, clusterOperator),
			operatorcli:   fakeoperator.NewSimpleClientset(ingressController),
		},
		{
			name: "Failed step run will not crash if it cannot get the clusterversions, clusteroperators, ingresscontrollers",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`running step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc]`),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc] encountered error: oh no!"),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal(`clusterversions.config.openshift.io "version" not found`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`github.com/Azure/ARO-RP/pkg/cluster.(*manager).logNodes-fm: null`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`github.com/Azure/ARO-RP/pkg/cluster.(*manager).logClusterOperators-fm: null`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`github.com/Azure/ARO-RP/pkg/cluster.(*manager).logIngressControllers-fm: null`),
				},
			},
			kubernetescli: fake.NewSimpleClientset(),
			configcli:     fakeconfig.NewSimpleClientset(),
			operatorcli:   fakeoperator.NewSimpleClientset(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := testlog.New()
			m := &manager{
				log:           log,
				kubernetescli: tt.kubernetescli,
				configcli:     tt.configcli,
				operatorcli:   tt.operatorcli,
			}

			err := m.runSteps(ctx, tt.steps)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			err = testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
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

			m := &manager{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				deployments: deploymentsClient,
			}

			err := m.deployARMTemplate(ctx, resourceGroup, "test", armTemplate, params)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestUpdateProvisionedBy(t *testing.T) {
	ctx := context.Background()
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

	openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
	fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
	fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateCreating,
			},
		},
	})
	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	clusterdoc, err := openShiftClustersDatabase.Dequeue(ctx)
	if err != nil {
		t.Fatal(err)
	}

	i := &manager{
		doc: clusterdoc,
		db:  openShiftClustersDatabase,
	}
	err = i.updateProvisionedBy(ctx)
	if err != nil {
		t.Error(err)
	}

	// Check it was set to the correct value in the database
	updatedClusterDoc, err := openShiftClustersDatabase.Get(ctx, strings.ToLower(key))
	if err != nil {
		t.Fatal(err)
	}
	if updatedClusterDoc.OpenShiftCluster.Properties.ProvisionedBy != version.GitCommit {
		t.Error("version was not added")
	}
}
