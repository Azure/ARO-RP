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
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
)

//go:embed staticresources
var staticFiles embed.FS

func TestDeployCreateOrUpdateSetsOwnerReferences(t *testing.T) {
	setPullSpec := "MyMUOPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	clientFake := ctrlfake.NewClientBuilder().Build()
	hookClient := testclienthelper.NewHookingClient(clientFake)
	log := logrus.NewEntry(logrus.StandardLogger())

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

	hookClient = hookClient.WithPostCreateHook(func(obj client.Object) error {
		// Check that each list of OwnerReferences contains our controller
		errs := deep.Equal([]metav1.OwnerReference{expectedOwner}, obj.GetOwnerReferences())
		for _, e := range errs {
			t.Error(e)
		}
		return nil
	})

	deployer := NewDeployer(log, hookClient, staticFiles, "staticresources")
	err := deployer.CreateOrUpdate(context.Background(), cluster, &config.MUODeploymentConfig{Pullspec: setPullSpec})
	if err != nil {
		t.Error(err)
	}
}

func TestDeployDelete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tally := make(map[string]int)

	log := logrus.NewEntry(logrus.StandardLogger())
	ch := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().Build()).WithPreDeleteHook(testclienthelper.TallyCountsAndKey(tally))

	deployer := NewDeployer(log, ch, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err != nil {
		t.Error(err)
	}

	expected := map[string]int{
		"Deployment/openshift-managed-upgrade-operator/managed-upgrade-operator": 1,
		"Namespace//openshift-managed-upgrade-operator":                          1,
		"bogon/openshift-managed-upgrade-operator/bogerus":                       1,
	}

	for _, err := range deep.Equal(expected, tally) {
		t.Error(err)
	}

}

func TestDeployDeleteFailure(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	ch := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().Build()).
		WithPreDeleteHook(func(o client.Object) error {
			return errors.New("fail")
		})

	deployer := NewDeployer(log, ch, staticFiles, "staticresources")
	err := deployer.Remove(context.Background(), config.MUODeploymentConfig{})
	if err == nil {
		t.Error(err)
	}
	if err.Error() != "error removing resource:\nfail\nfail\nfail" {
		t.Error(err)
	}
}

func TestDeployIsReady(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
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

	deployer := NewDeployer(log, clientFake, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if !ready {
		t.Error("deployment is not seen as ready")
	}
}

func TestDeployIsReadyMissing(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	clientFake := ctrlfake.NewClientBuilder().Build()
	deployer := NewDeployer(log, clientFake, staticFiles, "staticresources")
	ready, err := deployer.IsReady(context.Background(), "openshift-managed-upgrade-operator", "managed-upgrade-operator")
	if err != nil {
		t.Error(err)
	}
	if ready {
		t.Error("deployment is wrongly seen as ready")
	}
}
