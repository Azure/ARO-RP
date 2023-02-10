package deployer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// files in staticresources are example files that are not used anywhere

import (
	"context"
	"embed"
	"errors"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

//go:embed staticresources
var staticFiles embed.FS

func TestDeployCreateOrUpdateSetsOwnerReferences(t *testing.T) {
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

	// the OwnerReference that we expect to be set on each object we Ensure
	trueValue := true
	truePtr := &trueValue
	expectedOwner := metav1.OwnerReference{
		APIVersion:         "aro.openshift.io/v1alpha1",
		Kind:               "Cluster",
		Name:               arov1alpha1.SingletonClusterName,
		UID:                cluster.UID,
		BlockOwnerDeletion: truePtr,
		Controller:         truePtr,
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

	deployer := NewDeployer(clientFake, dh, staticFiles, "staticresources")
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

	clientFake := ctrlfake.NewClientBuilder().Build()
	dh := mock_dynamichelper.NewMockInterface(controller)
	dh.EXPECT().EnsureDeleted(gomock.Any(), "Deployment", "openshift-managed-upgrade-operator", "managed-upgrade-operator").Return(nil)

	deployer := NewDeployer(clientFake, dh, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err != nil {
		t.Error(err)
	}
}

func TestDeployDeleteFailure(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	clientFake := ctrlfake.NewClientBuilder().Build()
	dh := mock_dynamichelper.NewMockInterface(controller)
	dh.EXPECT().EnsureDeleted(gomock.Any(), "Deployment", "openshift-managed-upgrade-operator", "managed-upgrade-operator").Return(errors.New("fail"))

	deployer := NewDeployer(clientFake, dh, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err == nil {
		t.Error(err)
	}
	if err.Error() != "error removing deployment:\nfail" {
		t.Error(err)
	}
}

func TestDeployIsReady(t *testing.T) {
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

	deployer := NewDeployer(clientFake, nil, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if !ready {
		t.Error("deployment is not seen as ready")
	}
}

func TestDeployIsReadyMissing(t *testing.T) {
	clientFake := ctrlfake.NewClientBuilder().Build()
	deployer := NewDeployer(clientFake, nil, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if ready {
		t.Error("deployment is wrongly seen as ready")
	}
}
