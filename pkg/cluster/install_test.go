package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func failingFunc(context.Context) error { return errors.New("oh no!") }

func successfulActionStep(context.Context) error { return nil }

func successfulConditionStep(context.Context) (bool, error) { return true, nil }

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
		configcli     *configfake.Clientset
		operatorcli   *operatorfake.Clientset
		runType       string
	}{
		{
			name: "Failed install step run will log cluster version, cluster operator status, VM serial logs, and ingress information if available",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`running step [Action pkg/cluster.failingFunc]`),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("step [Action pkg/cluster.failingFunc] encountered error: oh no!"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logClusterVersion:.*"name": "version"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logNodes:.*"name": "node"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logClusterOperators:.*"name": "operator"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logIngressControllers:.*"name": "ingress-controller"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pkg/cluster/failurediagnostics.(*manager).LogVMSerialConsole: vmclient missing`),
				},
			},
			kubernetescli: fake.NewSimpleClientset(node),
			configcli:     configfake.NewSimpleClientset(clusterVersion, clusterOperator),
			operatorcli:   operatorfake.NewSimpleClientset(ingressController),
			runType:       "install",
		},
		{
			name: "Failed update step run will log cluster version, cluster operator status, and ingress information if available",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`running step [Action pkg/cluster.failingFunc]`),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("step [Action pkg/cluster.failingFunc] encountered error: oh no!"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logClusterVersion:.*"name": "version"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logNodes:.*"name": "node"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logClusterOperators:.*"name": "operator"`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`(?s)pkg/cluster.\(\*manager\).logIngressControllers:.*"name": "ingress-controller"`),
				},
			},
			kubernetescli: fake.NewSimpleClientset(node),
			configcli:     configfake.NewSimpleClientset(clusterVersion, clusterOperator),
			operatorcli:   operatorfake.NewSimpleClientset(ingressController),
			runType:       "update",
		},
		{
			name: "Failed install step run will not crash if it cannot get the clusterversions, clusteroperators, ingresscontrollers",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "oh no!",
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`running step [Action pkg/cluster.failingFunc]`),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("step [Action pkg/cluster.failingFunc] encountered error: oh no!"),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal(`clusterversions.config.openshift.io "version" not found`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pkg/cluster.(*manager).logNodes: null`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pkg/cluster.(*manager).logClusterOperators: null`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pkg/cluster.(*manager).logIngressControllers: null`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pkg/cluster/failurediagnostics.(*manager).LogVMSerialConsole: vmclient missing`),
				},
			},
			kubernetescli: fake.NewSimpleClientset(),
			configcli:     configfake.NewSimpleClientset(),
			operatorcli:   operatorfake.NewSimpleClientset(),
			runType:       "install",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			h, log := testlog.New()
			m := &manager{
				log:           log,
				kubernetescli: tt.kubernetescli,
				configcli:     tt.configcli,
				operatorcli:   tt.operatorcli,
				now:           func() time.Time { return time.Now() },
			}

			err := m.runSteps(ctx, tt.steps, tt.runType)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			err = testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
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

	updatedClusterDoc, err := openShiftClustersDatabase.Get(ctx, strings.ToLower(key))
	if err != nil {
		t.Fatal(err)
	}
	if updatedClusterDoc.OpenShiftCluster.Properties.ProvisionedBy != version.GitCommit {
		t.Error("version was not added")
	}
}

func TestInstallationTimeMetrics(t *testing.T) {
	_, log := testlog.New()

	for _, tt := range []struct {
		name          string
		metricsTopic  string
		timePerStep   int64
		steps         []steps.Step
		wantedMetrics []testmonitor.ExpectedMetric
	}{
		{
			name:         "Failed step run will not generate any install time metrics",
			metricsTopic: "install",
			steps: []steps.Step{
				steps.Action(successfulActionStep),
				steps.Action(failingFunc),
			},
		},
		{
			name:         "Succeeded step run for cluster installation will generate a valid install time metrics",
			metricsTopic: "install",
			timePerStep:  2,
			steps: []steps.Step{
				steps.Action(successfulActionStep),
				steps.Condition(successfulConditionStep, 30*time.Minute, true),
				steps.Action(successfulActionStep),
			},
			wantedMetrics: []testmonitor.ExpectedMetric{
				testmonitor.Metric("backend.openshiftcluster.install.duration.total.seconds", int64(4), nil),
				testmonitor.Metric("backend.openshiftcluster.install.action.successfulActionStep.duration.seconds", int64(2), nil),
				testmonitor.Metric("backend.openshiftcluster.install.condition.successfulConditionStep.duration.seconds", int64(2), nil),
			},
		},
		{
			name:         "Succeeded step run for cluster update will generate a valid install time metrics",
			metricsTopic: "update",
			timePerStep:  3,
			steps: []steps.Step{
				steps.Action(successfulActionStep),
				steps.Condition(successfulConditionStep, 30*time.Minute, true),
				steps.Action(successfulActionStep),
			},
			wantedMetrics: []testmonitor.ExpectedMetric{
				testmonitor.Metric("backend.openshiftcluster.update.duration.total.seconds", int64(6), nil),
				testmonitor.Metric("backend.openshiftcluster.update.action.successfulActionStep.duration.seconds", int64(3), nil),
				testmonitor.Metric("backend.openshiftcluster.update.condition.successfulConditionStep.duration.seconds", int64(3), nil),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fm := testmonitor.NewFakeEmitter(t)
			m := &manager{
				log:            log,
				metricsEmitter: fm,
				now:            func() time.Time { return time.Now().Add(time.Duration(tt.timePerStep) * time.Second) },
			}

			m.runSteps(ctx, tt.steps, tt.metricsTopic)
			fm.VerifyEmittedMetrics(tt.wantedMetrics...)
		})
	}
}

func TestRunHiveInstallerSetsCreatedByHiveFieldToTrueInClusterDoc(t *testing.T) {
	ctx := context.Background()
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

	openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
	fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
	doc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateCreating,
			},
		},
	}

	fixture.AddOpenShiftClusterDocuments(doc)
	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	dequeuedDoc, err := openShiftClustersDatabase.Dequeue(ctx)
	if err != nil {
		t.Fatal(err)
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	hiveClusterManagerMock := mock_hive.NewMockClusterManager(controller)
	hiveClusterManagerMock.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	m := &manager{
		doc: dequeuedDoc,
		db:  openShiftClustersDatabase,
		openShiftClusterDocumentVersioner: &FakeOpenShiftClusterDocumentVersionerService{
			expectedOpenShiftVersion: nil,
			expectedError:            nil,
		},
		hiveClusterManager: hiveClusterManagerMock,
	}

	err = m.runHiveInstaller(ctx)
	if err != nil {
		t.Fatal(err)
	}

	updatedDoc, err := openShiftClustersDatabase.Get(ctx, strings.ToLower(key))
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	got := updatedDoc.OpenShiftCluster.Properties.HiveProfile.CreatedByHive
	if got != expected {
		t.Fatalf("expected updatedDoc.OpenShiftCluster.Properties.HiveProfile.CreatedByHive set to %v, but got %v", expected, got)
	}
}
