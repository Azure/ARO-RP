package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

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
		configcli     *configfake.Clientset
		operatorcli   *operatorfake.Clientset
	}{
		{
			name: "Failed step run will log cluster version, cluster operator status, and ingress information if available",
			steps: []steps.Step{
				steps.Action(failingFunc),
			},
			wantErr: "step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc] encountered error: oh no!",
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
			wantErr: "step [Action github.com/Azure/ARO-RP/pkg/cluster.failingFunc] encountered error: oh no!",
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
