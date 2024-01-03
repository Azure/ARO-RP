package kubernetes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// Restart restarts the given Deployment by performing the equivalent of a `kubectl rollout restart deployment <deployment-name>` against it.
// Note that Restart needs the namespace of the Deployment as a separate argument even though the namespace is already "encapsulated"
// by the DeploymentInterface (and the namespace you pass should be the same one used to create the DeploymentInterface.
func Restart(ctx context.Context, cli appsv1client.DeploymentInterface, namespace string, deploymentName string) error {
	dac := appsv1.Deployment(deploymentName, namespace).WithSpec(
		appsv1.DeploymentSpec().WithTemplate(
			corev1.PodTemplateSpec().WithAnnotations(map[string]string{
				"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
			}),
		),
	)
	_, err := cli.Apply(ctx, dac, metav1.ApplyOptions{FieldManager: "aro-rp", Force: true})
	return err
}
