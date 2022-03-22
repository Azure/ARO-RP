package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	k8scli := fake.NewSimpleClientset()
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

	deployer := newDeployer(k8scli, dh)
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

func TestDeployCreateOrUpdateSetsOwnerReferences(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	k8scli := fake.NewSimpleClientset()
	dh := mock_dynamichelper.NewMockInterface(controller)

	// the OwnerReference that we expect to be set on each object we Ensure
	pointerToTrueForSomeReason := bool(true)
	expectedOwner := metav1.OwnerReference{
		APIVersion:         "aro.openshift.io/v1alpha1",
		Kind:               "Cluster",
		Name:               arov1alpha1.SingletonClusterName,
		UID:                cluster.UID,
		BlockOwnerDeletion: &pointerToTrueForSomeReason,
		Controller:         &pointerToTrueForSomeReason,
	}

	// save the list of OwnerReferences on each of the Ensured objects
	var ownerReferences [][]metav1.OwnerReference
	check := func(ctx context.Context, objs ...kruntime.Object) error {
		for _, i := range objs {
			obj, err := meta.Accessor(i)
			if err != nil {
				return err
			}
			ownerReferences = append(ownerReferences, obj.GetOwnerReferences())
		}
		return nil
	}
	dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Do(check).Return(nil)

	deployer := newDeployer(k8scli, dh)
	err := deployer.CreateOrUpdate(context.Background(), cluster, &config.MUODeploymentConfig{Pullspec: setPullSpec})
	if err != nil {
		t.Error(err)
	}

	// Check that each list of OwnerReferences contains our controller
	for _, references := range ownerReferences {
		errs := deep.Equal([]metav1.OwnerReference{expectedOwner}, references)
		for _, e := range errs {
			t.Error(e)
		}
	}
}

func TestDeployDelete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	k8scli := fake.NewSimpleClientset()
	dh := mock_dynamichelper.NewMockInterface(controller)
	dh.EXPECT().EnsureDeleted(gomock.Any(), "Deployment", "openshift-managed-upgrade-operator", "managed-upgrade-operator").Return(nil)

	deployer := newDeployer(k8scli, dh)
	err := deployer.Remove(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestDeployDeleteFailure(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	k8scli := fake.NewSimpleClientset()
	dh := mock_dynamichelper.NewMockInterface(controller)
	dh.EXPECT().EnsureDeleted(gomock.Any(), "Deployment", "openshift-managed-upgrade-operator", "managed-upgrade-operator").Return(errors.New("fail"))

	deployer := newDeployer(k8scli, dh)
	err := deployer.Remove(context.Background())
	if err == nil {
		t.Error(err)
	}
	if err.Error() != "error removing MUO:\nfail" {
		t.Error(err)
	}
}

func TestDeployIsReady(t *testing.T) {
	specReplicas := int32(1)
	k8scli := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "managed-upgrade-operator",
			Namespace:  "openshift-managed-upgrade-operator",
			Generation: 1234,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &specReplicas,
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration:  1234,
			Replicas:            1,
			ReadyReplicas:       1,
			UpdatedReplicas:     1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})

	deployer := newDeployer(k8scli, nil)
	ready, err := deployer.IsReady(context.Background())
	if err != nil {
		t.Error(err)
	}
	if !ready {
		t.Error("deployment is not seen as ready")
	}
}

func TestDeployIsReadyMissing(t *testing.T) {
	k8scli := fake.NewSimpleClientset()
	deployer := newDeployer(k8scli, nil)
	ready, err := deployer.IsReady(context.Background())
	if err != nil {
		t.Error(err)
	}
	if ready {
		t.Error("deployment is wrongly seen as ready")
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
		expected         []string
	}{
		{
			name:             "local",
			deploymentConfig: &config.MUODeploymentConfig{EnableConnected: false},
			expected: []string{
				"configManager:",
				"  localConfigName: managed-upgrade-config",
				"  source: LOCAL",
				"  watchInterval: 1",
				"healthCheck:",
				"  ignoredCriticals:",
				"  - PrometheusRuleFailures",
				"  - CannotRetrieveUpdates",
				"  - FluentdNodeDown",
				"  ignoredNamespaces:",
				"  - openshift-logging",
				"  - openshift-redhat-marketplace",
				"  - openshift-operators",
				"  - openshift-user-workload-monitoring",
				"  - openshift-pipelines",
				"  - openshift-azure-logging",
				"maintenance:",
				"  controlPlaneTime: 90",
				"  ignoredAlerts:",
				"    controlPlaneCriticals:",
				"    - ClusterOperatorDown",
				"    - ClusterOperatorDegraded",
				"nodeDrain:",
				"  expectedNodeDrainTime: 8",
				"  timeOut: 45",
				"scale:",
				"  timeOut: 30",
				"upgradeWindow:",
				"  delayTrigger: 30",
				"  timeOut: 120",
				"",
			},
		},
		{
			name:             "connected",
			deploymentConfig: &config.MUODeploymentConfig{EnableConnected: true, OCMBaseURL: "https://example.com"},
			expected: []string{
				"configManager:",
				"  ocmBaseUrl: https://example.com",
				"  source: OCM",
				"  watchInterval: 1",
				"healthCheck:",
				"  ignoredCriticals:",
				"  - PrometheusRuleFailures",
				"  - CannotRetrieveUpdates",
				"  - FluentdNodeDown",
				"  ignoredNamespaces:",
				"  - openshift-logging",
				"  - openshift-redhat-marketplace",
				"  - openshift-operators",
				"  - openshift-user-workload-monitoring",
				"  - openshift-pipelines",
				"  - openshift-azure-logging",
				"maintenance:",
				"  controlPlaneTime: 90",
				"  ignoredAlerts:",
				"    controlPlaneCriticals:",
				"    - ClusterOperatorDown",
				"    - ClusterOperatorDegraded",
				"nodeDrain:",
				"  expectedNodeDrainTime: 8",
				"  timeOut: 45",
				"scale:",
				"  timeOut: 30",
				"upgradeWindow:",
				"  delayTrigger: 30",
				"  timeOut: 120",
				"",
			},
		},
	}
	for _, tt := range tests {
		k8scli := fake.NewSimpleClientset()
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

		deployer := newDeployer(k8scli, dh)
		err := deployer.CreateOrUpdate(context.Background(), cluster, tt.deploymentConfig)
		if err != nil {
			t.Error(err)
		}

		foundConfig := false
		for _, cms := range configs {
			if cms.Name == "managed-upgrade-operator-config" && cms.Namespace == "openshift-managed-upgrade-operator" {
				foundConfig = true
				errs := deep.Equal(tt.expected, strings.Split(cms.Data["config.yaml"], "\n"))
				for _, e := range errs {
					t.Error(e)
				}
			}
		}

		if !foundConfig {
			t.Error("MUO config was not found")
		}
	}
}
