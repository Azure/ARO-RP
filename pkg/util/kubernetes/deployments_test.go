package kubernetes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
	"github.com/Azure/ARO-RP/test/util/serversideapply"
)

func TestRestart(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                string
		deployment          *appsv1.Deployment
		deploymentCli       func(*fake.Clientset) appsv1client.DeploymentInterface
		deploymentName      string
		deploymentNamespace string
		wantErrMsg          string
	}{
		{
			name: "Success",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-operator-master",
					Namespace: "openshift-azure-operator",
				},
			},
			deploymentCli: func(clientset *fake.Clientset) appsv1client.DeploymentInterface {
				return clientset.AppsV1().Deployments("openshift-azure-operator")
			},
			deploymentName:      "aro-operator-master",
			deploymentNamespace: "openshift-azure-operator",
		},
		{
			name: "Caller passed a DeploymentInterface whose namespace doesn't match the one passed",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-operator-master",
					Namespace: "openshift-azure-operator",
				},
			},
			deploymentCli: func(clientset *fake.Clientset) appsv1client.DeploymentInterface {
				return clientset.AppsV1().Deployments("azure-operator")
			},
			deploymentName:      "aro-operator-master",
			deploymentNamespace: "openshift-azure-operator",
			wantErrMsg:          `request namespace does not match object namespace, request: "azure-operator" object: "openshift-azure-operator"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientset := serversideapply.CliWithApply([]string{"deployments"}, tt.deployment)
			err := Restart(ctx, tt.deploymentCli(clientset), tt.deploymentNamespace, tt.deploymentName)
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)

			d, _ := clientset.AppsV1().Deployments(tt.deploymentNamespace).Get(ctx, tt.deploymentName, metav1.GetOptions{})

			// Checking for this annotation to be here is consistent with how Kubernetes really behaves;
			// even after the Deployment is done with the restart, the annotation remains.
			if err == nil {
				foundRestartAnnotation := false
				if d.Spec.Template.Annotations != nil {
					for a := range d.Spec.Template.Annotations {
						if a == "kubectl.kubernetes.io/restartedAt" {
							foundRestartAnnotation = true
							break
						}
					}
				}

				if !foundRestartAnnotation {
					t.Errorf("Expected restart annotation is missing from Deployment")
				}
			}
		})
	}
}
