package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func failingFunc(context.Context) error { return errors.New("oh no!") }

func successfulActionStep(context.Context) error { return nil }

func successfulConditionStep(context.Context) (bool, error) { return true, nil }

type fakeMetricsEmitter struct {
	Metrics map[string]int64
}

func newfakeMetricsEmitter() *fakeMetricsEmitter {
	m := make(map[string]int64)
	return &fakeMetricsEmitter{
		Metrics: m,
	}
}

func (e *fakeMetricsEmitter) EmitGauge(topic string, value int64, dims map[string]string) {
	e.Metrics[topic] = value
}

func (e *fakeMetricsEmitter) EmitFloat(topic string, value float64, dims map[string]string) {}

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
			configcli:     configfake.NewSimpleClientset(clusterVersion, clusterOperator),
			operatorcli:   operatorfake.NewSimpleClientset(ingressController),
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
			configcli:     configfake.NewSimpleClientset(),
			operatorcli:   operatorfake.NewSimpleClientset(),
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

			err := m.runSteps(ctx, tt.steps, false)
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
	fm := newfakeMetricsEmitter()

	for _, tt := range []struct {
		name          string
		steps         []steps.Step
		wantedMetrics map[string]int64
	}{
		{
			name: "Failed step run will not generate any install time metrics",
			steps: []steps.Step{
				steps.Action(successfulActionStep),
				steps.Action(failingFunc),
			},
		},
		{
			name: "Succeeded step run will generate a valid install time metrics",
			steps: []steps.Step{
				steps.Action(successfulActionStep),
				steps.Condition(successfulConditionStep, 30*time.Minute, true),
				steps.AuthorizationRefreshingAction(nil, steps.Action(successfulActionStep)),
			},
			wantedMetrics: map[string]int64{
				"backend.openshiftcluster.installtime.total":                             6,
				"backend.openshiftcluster.installtime.action.successfulActionStep":       2,
				"backend.openshiftcluster.installtime.condition.successfulConditionStep": 2,
				"backend.openshiftcluster.installtime.refreshing.successfulActionStep":   2,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &manager{
				log:            log,
				metricsEmitter: fm,
				now:            func() time.Time { return time.Now().Add(2 * time.Second) },
			}

			err := m.runSteps(ctx, tt.steps, true)
			if err != nil {
				if len(fm.Metrics) != 0 {
					t.Error("fake metrics obj should be empty when run steps failed")
				}
			} else {
				if tt.wantedMetrics != nil {
					for k, v := range tt.wantedMetrics {
						time, ok := fm.Metrics[k]
						if !ok {
							t.Errorf("unexpected metrics topic: %s", k)
						}
						if time != v {
							t.Errorf("incorrect fake metrics obj, want: %d, got: %d", v, time)
						}
					}
				}
			}
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
	hiveClusterManagerMock.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

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
