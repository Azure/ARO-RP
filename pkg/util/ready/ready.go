package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// NodeIsReady returns true if a Node is considered ready
func NodeIsReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// PodIsReady returns true if a Pod is considered ready
func PodIsReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// DaemonSetIsReady returns true if a DaemonSet is considered ready
func DaemonSetIsReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable &&
		ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
		ds.Generation == ds.Status.ObservedGeneration
}

// CheckDaemonSetIsReady returns a function which polls a DaemonSet and returns
// its readiness
func CheckDaemonSetIsReady(cli appsv1client.DaemonSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ds, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DaemonSetIsReady(ds), nil
	}
}

// DeploymentIsReady returns true if a Deployment is considered ready
func DeploymentIsReady(d *appsv1.Deployment) bool {
	specReplicas := int32(1)
	if d.Spec.Replicas != nil {
		specReplicas = *d.Spec.Replicas
	}

	return specReplicas == d.Status.AvailableReplicas &&
		specReplicas == d.Status.UpdatedReplicas &&
		d.Generation == d.Status.ObservedGeneration
}

// CheckDeploymentIsReady returns a function which polls a Deployment and
// returns its readiness
func CheckDeploymentIsReady(cli appsv1client.DeploymentInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		d, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DeploymentIsReady(d), nil
	}
}
