package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	_ "embed"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
)

//go:embed test_files/local.yaml
var expectedLocalConfig []byte

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	clientFake := ctrlfake.NewClientBuilder().Build()
	log := logrus.NewEntry(logrus.StandardLogger())
	deployedObjects := make(map[string]int)

	// When the DynamicHelper is called, count the number of objects it creates
	// and capture any deployments so that we can check the pullspec
	var deployments []*appsv1.Deployment
	ch := testclienthelper.NewHookingClient(clientFake).
		WithPostCreateHook(testclienthelper.TallyCounts(deployedObjects)).
		WithPostCreateHook(func(o client.Object) error {
			if d, ok := o.(*appsv1.Deployment); ok {
				deployments = append(deployments, d)
			}
			return nil
		})

	deployer := deployer.NewDeployer(log, ch, staticFiles, "staticresources")
	err := deployer.CreateOrUpdate(context.Background(), cluster, &config.MUODeploymentConfig{Pullspec: setPullSpec})
	if err != nil {
		t.Error(err)
	}

	// We expect these numbers of resources to be created
	expectedKinds := map[string]int{
		"ClusterRole":              1,
		"ConfigMap":                2,
		"ClusterRoleBinding":       1,
		"CustomResourceDefinition": 1,
		"Deployment":               1,
		"Namespace":                1,
		"Role":                     5,
		"RoleBinding":              5,
		"ServiceAccount":           1,
		"PrometheusRule":           1,
	}
	errs := deep.Equal(deployedObjects, expectedKinds)
	for _, e := range errs {
		t.Error(e)
	}

	// Ensure we have set the pullspec set on the containers
	for _, d := range deployments {
		for _, c := range d.Spec.Template.Spec.Containers {
			if c.Image != setPullSpec {
				t.Errorf("expected %s, got %s for pullspec", setPullSpec, c.Image)
			}
		}
	}
}

func TestDeployConfig(t *testing.T) {
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	tests := []struct {
		name             string
		deploymentConfig *config.MUODeploymentConfig
		expected         []byte
	}{
		{
			name:             "local",
			deploymentConfig: &config.MUODeploymentConfig{},
			expected:         expectedLocalConfig,
		},
	}
	for _, tt := range tests {
		clientFake := ctrlfake.NewClientBuilder().Build()
		log := logrus.NewEntry(logrus.StandardLogger())

		// When the DynamicHelper is called, capture configmaps to inspect them
		var configs []*corev1.ConfigMap

		ch := testclienthelper.NewHookingClient(clientFake).
			WithPostCreateHook(func(o client.Object) error {
				if cm, ok := o.(*corev1.ConfigMap); ok {
					configs = append(configs, cm)
				}
				return nil
			})

		deployer := deployer.NewDeployer(log, ch, staticFiles, "staticresources")
		err := deployer.CreateOrUpdate(context.Background(), cluster, tt.deploymentConfig)
		if err != nil {
			t.Error(err)
		}

		foundConfig := false
		for _, cms := range configs {
			if cms.Name == "managed-upgrade-operator-config" && cms.Namespace == "openshift-managed-upgrade-operator" {
				foundConfig = true

				expectedMap := make(map[string]interface{})
				yaml.Unmarshal(tt.expected, &expectedMap)

				resultMap := make(map[string]interface{})
				yaml.Unmarshal([]byte(cms.Data["config.yaml"]), &resultMap)

				err := deep.Equal(expectedMap, resultMap)
				if err != nil {
					t.Error(err)
				}
			}
		}

		if !foundConfig {
			t.Error("MUO config was not found")
		}
	}
}
