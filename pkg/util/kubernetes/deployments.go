package kubernetes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// Restart restarts the given Deployment by performing the equivalent of a `kubectl rollout restart deployment <deployment-name>` against it.
func Restart(ctx context.Context, cli appsv1client.DeploymentInterface, deploymentName string) error {
	d, err := cli.Get(ctx, deploymentName, metav1.GetOptions{})

	if err != nil {
		return err
	}

	if d.Spec.Template.ObjectMeta.Annotations == nil {
		d.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}

	d.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = cli.Update(ctx, d, metav1.UpdateOptions{})

	if err != nil {
		return err
	}

	return nil
}
