package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"
	"testing"

	"github.com/go-test/deep"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdh "github.com/Azure/ARO-RP/test/util/dynamichelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestDeployCreateOrUpdateCorrectKinds(t *testing.T) {
	_, log := testlog.New()

	setPullSpec := "MyGuardRailsPullSpec"
	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}
	ver, err := version.ParseVersion("4.11.0")
	if err != nil {
		t.Fatal(err)
	}
	deployConfig := getDefaultDeployConfig(context.Background(), cluster, ver)
	deployConfig.Pullspec = setPullSpec

	deployedObjects := map[string]int{}
	wrappedClient := testdh.NewRedirectingClient(ctrlfake.NewClientBuilder().Build()).
		WithCreateHook(testdh.TallyCounts(deployedObjects))

	dh := dynamichelper.NewWithClient(log, wrappedClient)
	deployer := deployer.NewDeployer(dh, staticFiles, "staticresources")
	err = deployer.CreateOrUpdate(context.Background(), cluster, deployConfig)
	if err != nil {
		t.Error(err)
	}

	// We expect these numbers of resources to be created
	expectedKinds := map[string]int{
		"ClusterRole":                    1,
		"ClusterRoleBinding":             1,
		"CustomResourceDefinition":       10,
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
