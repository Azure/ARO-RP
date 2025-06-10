package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	_ "embed"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	setPullSpec := "MyGuardRailsPullSpec"
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
	err := deployer.CreateOrUpdate(context.Background(), cluster, &config.GuardRailsDeploymentConfig{Pullspec: setPullSpec})
	if err != nil {
		t.Error(err)
	}

	// We expect these numbers of resources to be created
	expectedKinds := map[string]int{
		"ClusterRole":                    1,
		"ClusterRoleBinding":             1,
		"CustomResourceDefinition":       13,
		"Deployment":                     2,
		"Namespace":                      1,
		"Role":                           1,
		"RoleBinding":                    1,
		"ServiceAccount":                 1,
		"MutatingWebhookConfiguration":   1,
		"PodDisruptionBudget":            1,
		"Service":                        1,
		"Secret":                         1,
		"ResourceQuota":                  1,
		"ValidatingWebhookConfiguration": 1,
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
