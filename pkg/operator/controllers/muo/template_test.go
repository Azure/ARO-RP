package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/go-test/deep"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/test/util/kubetest"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

//go:embed test_files/local.yaml
var expectedLocalConfig []byte

//go:embed test_files/connected.yaml
var expectedConnectedConfig []byte

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	_, log := testlog.New()

	setPullSpec := "MyGuardRailsPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	deployedObjects := map[string]int{}
	wrappedClient := kubetest.NewRedirectingClient(ctrlfake.NewClientBuilder().Build()).
		WithCreateHook(kubetest.TallyCounts(deployedObjects))
	dh := dynamichelper.NewWithClient(log, wrappedClient)

	deployer := deployer.NewDeployer(dh, staticFiles, "staticresources")
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
		"Role":                     4,
		"RoleBinding":              4,
		"ServiceAccount":           1,
	}

	for _, v := range deep.Equal(deployedObjects, expectedKinds) {
		t.Errorf("created does not match: %s", v)
	}

	deployments := &appsv1.DeploymentList{}
	err = wrappedClient.List(context.Background(), deployments)
	if err != nil {
		t.Error(err)
	}

	// Ensure we have set the pullspec set on the containers
	for _, d := range deployments.Items {
		for _, c := range d.Spec.Template.Spec.Containers {
			if c.Image != setPullSpec {
				t.Errorf("expected %s, got %s for pullspec", setPullSpec, c.Image)
			}
		}
	}
}

func TestDeployConfig(t *testing.T) {
	_, log := testlog.New()

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
			deploymentConfig: &config.MUODeploymentConfig{EnableConnected: false},
			expected:         expectedLocalConfig,
		},
		{
			name:             "connected",
			deploymentConfig: &config.MUODeploymentConfig{EnableConnected: true, OCMBaseURL: "https://example.com"},
			expected:         expectedConnectedConfig,
		},
	}
	for _, tt := range tests {
		wrappedClient := kubetest.NewRedirectingClient(ctrlfake.NewClientBuilder().Build())
		dh := dynamichelper.NewWithClient(log, wrappedClient)

		deployer := deployer.NewDeployer(dh, staticFiles, "staticresources")
		err := deployer.CreateOrUpdate(context.Background(), cluster, tt.deploymentConfig)
		if err != nil {
			t.Error(err)
		}

		configs := &corev1.ConfigMapList{}
		err = wrappedClient.List(context.Background(), configs)
		if err != nil {
			t.Error(err)
		}

		foundConfig := false
		for _, cms := range configs.Items {
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
