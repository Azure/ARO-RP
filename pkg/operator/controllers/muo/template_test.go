package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

//go:embed test_files/local.yaml
var expectedLocalConfig []byte

//go:embed test_files/connected.yaml
var expectedConnectedConfig []byte

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	clientFake := ctrlfake.NewClientBuilder().Build()
	dh := mock_dynamichelper.NewMockInterface(controller)

	// When the DynamicHelper is called, count the number of objects it creates
	// and capture any deployments so that we can check the pullspec
	var deployments []*appsv1.Deployment
	deployedObjects := make(map[string]int)
	check := func(ctx context.Context, objs ...kruntime.Object) error {
		m := meta.NewAccessor()
		for _, i := range objs {
			kind, err := m.Kind(i)
			if err != nil {
				return err
			}
			if d, ok := i.(*appsv1.Deployment); ok {
				deployments = append(deployments, d)
			}
			deployedObjects[kind] = deployedObjects[kind] + 1
		}
		return nil
	}
	dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Do(check).Return(nil)

	deployer := deployer.NewDeployer(clientFake, dh, staticFiles, "staticresources")
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
	controller := gomock.NewController(t)
	defer controller.Finish()

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
		clientFake := ctrlfake.NewClientBuilder().Build()
		dh := mock_dynamichelper.NewMockInterface(controller)

		// When the DynamicHelper is called, capture configmaps to inspect them
		var configs []*corev1.ConfigMap
		check := func(ctx context.Context, objs ...kruntime.Object) error {
			for _, i := range objs {
				if cm, ok := i.(*corev1.ConfigMap); ok {
					configs = append(configs, cm)
				}
			}
			return nil
		}
		dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Do(check).Return(nil)

		deployer := deployer.NewDeployer(clientFake, dh, staticFiles, "staticresources")
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
