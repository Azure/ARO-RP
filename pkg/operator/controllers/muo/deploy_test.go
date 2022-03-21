package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
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
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerPullSpec: setPullSpec,
			},
		},
	}

	k8scli := fake.NewSimpleClientset()
	dh := mock_dynamichelper.NewMockInterface(controller)

	// When the DynamicHelper is called, count the number of objects it creates
	// and capture any deployments so that we can check the pull secret
	var deployments []*appsv1.Deployment
	deployedObjects := make(map[string]int)
	check := func(ctx context.Context, objs ...kruntime.Object) error {
		m := meta.NewAccessor()
		for _, i := range objs {
			kind, err := m.Kind(i)
			if err != nil {
				return err
			}
			deployedObjects[kind] = deployedObjects[kind] + 1
		}
		return nil
	}
	dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Do(check).Return(nil)

	deployer := newDeployer(k8scli, dh)
	err := deployer.CreateOrUpdate(context.Background(), cluster)
	if err != nil {
		t.Error(err)
	}

	// We expect these numbers of resources to be created
	expectedKinds := map[string]int{
		"ClusterRole":              1,
		"ConfigMap":                1,
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
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerPullSpec: setPullSpec,
			},
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
	err := deployer.CreateOrUpdate(context.Background(), cluster)
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
