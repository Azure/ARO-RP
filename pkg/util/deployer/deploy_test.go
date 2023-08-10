package deployer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// files in staticresources are example files that are not used anywhere

import (
	"context"
	"embed"
	"errors"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	testdh "github.com/Azure/ARO-RP/test/util/dynamichelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

//go:embed staticresources
var staticFiles embed.FS

func TestDeployCreateOrUpdateSetsOwnerReferences(t *testing.T) {
	_, log := testlog.New()

	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}
	deployedObjects := map[string]int{}
	wrappedClient := testdh.NewRedirectingClient(ctrlfake.NewClientBuilder().Build()).
		WithCreateHook(
			func(obj client.Object) error {
				m := meta.NewAccessor()
				kind, err := m.Kind(obj)
				if err != nil {
					return err
				}

				deployedObjects[kind] += 1
				return nil
			})

	dh := dynamichelper.NewWithClient(log, wrappedClient)

	// the OwnerReference that we expect to be set on each object we Ensure
	expectedOwner := metav1.OwnerReference{
		APIVersion:         "aro.openshift.io/v1alpha1",
		Kind:               "Cluster",
		Name:               arov1alpha1.SingletonClusterName,
		UID:                cluster.UID,
		BlockOwnerDeletion: to.BoolPtr(true),
		Controller:         to.BoolPtr(true),
	}

	deployer := NewDeployer(dh, staticFiles, "staticresources")
	err := deployer.CreateOrUpdate(context.Background(), cluster, &config.MUODeploymentConfig{Pullspec: setPullSpec})
	if err != nil {
		t.Error(err)
	}

	// We expect these numbers of resources to be created
	expectedKinds := map[string]int{
		"Deployment": 1,
	}
	errs := deep.Equal(deployedObjects, expectedKinds)
	for _, e := range errs {
		t.Error(e)
	}

	deployments := &appsv1.DeploymentList{}
	err = wrappedClient.List(context.Background(), deployments)
	if err != nil {
		t.Error(err)
	}
	for _, v := range deployments.Items {
		errs := deep.Equal([]metav1.OwnerReference{expectedOwner}, v.GetOwnerReferences())
		for _, e := range errs {
			t.Error(e)
		}
	}
}

func TestDeployDelete(t *testing.T) {
	_, log := testlog.New()
	dh := dynamichelper.NewWithClient(log, ctrlfake.NewClientBuilder().Build())

	deployer := NewDeployer(dh, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err != nil {
		t.Error(err)
	}
}

func TestDeployDeleteFailure(t *testing.T) {
	_, log := testlog.New()
	wrappedClient := testdh.NewRedirectingClient(ctrlfake.NewClientBuilder().Build()).
		WithDeleteHook(
			func(obj client.Object) error {
				return errors.New("fail")
			})
	dh := dynamichelper.NewWithClient(log, wrappedClient)

	deployer := NewDeployer(dh, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err == nil {
		t.Error(err)
	}
	if err.Error() != "error removing resource:\nfail" {
		t.Error(err)
	}
}

func TestDeployIsReady(t *testing.T) {
	_, log := testlog.New()
	specReplicas := int32(1)
	clientFake := ctrlfake.NewClientBuilder().WithObjects(&appsv1.Deployment{
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
	}).Build()
	dh := dynamichelper.NewWithClient(log, clientFake)

	deployer := NewDeployer(dh, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if !ready {
		t.Error("deployment is not seen as ready")
	}
}

func TestDeployIsReadyMissing(t *testing.T) {
	_, log := testlog.New()
	dh := dynamichelper.NewWithClient(log, ctrlfake.NewClientBuilder().Build())
	deployer := NewDeployer(dh, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if ready {
		t.Error("deployment is wrongly seen as ready")
	}
}
