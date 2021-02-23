package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	machineconfigurationv1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
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

// DaemonSetIsReady returns true if a DaemonSet is considered ready
func DaemonSetIsReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable &&
		ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
		ds.Generation == ds.Status.ObservedGeneration
}

// ServiceIsReady returns true if a Service is considered ready
func ServiceIsReady(svc *corev1.Service) bool {
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) <= 0 {
			return false
		}
	case corev1.ServiceTypeClusterIP:
		if net.ParseIP(svc.Spec.ClusterIP) == nil {
			return false
		}
	default:
		return false
	}
	return true
}

// CheckDaemonSetIsReady returns a function which polls a DaemonSet and returns
// its readiness
func CheckDaemonSetIsReady(ctx context.Context, cli appsv1client.DaemonSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ds, err := cli.Get(ctx, name, metav1.GetOptions{})
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
		specReplicas == d.Status.Replicas &&
		d.Generation == d.Status.ObservedGeneration
}

// CheckDeploymentIsReady returns a function which polls a Deployment and
// returns its readiness
func CheckDeploymentIsReady(ctx context.Context, cli appsv1client.DeploymentInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		d, err := cli.Get(ctx, name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DeploymentIsReady(d), nil
	}
}

// PodIsRunning returns true if a Pod is running
func PodIsRunning(p *corev1.Pod) bool {
	return p.Status.Phase == corev1.PodRunning
}

// CheckPodIsRunning returns a function which polls a Pod and returns if it is
// running
func CheckPodIsRunning(ctx context.Context, cli corev1client.PodInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		p, err := cli.Get(ctx, name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return PodIsRunning(p), nil
	}
}

// StatefulSetIsReady returns true if a StatefulSet is considered ready
func StatefulSetIsReady(s *appsv1.StatefulSet) bool {
	specReplicas := int32(1)
	if s.Spec.Replicas != nil {
		specReplicas = *s.Spec.Replicas
	}

	return specReplicas == s.Status.ReadyReplicas &&
		specReplicas == s.Status.UpdatedReplicas &&
		s.Generation == s.Status.ObservedGeneration
}

// CheckStatefulSetIsReady returns a function which polls a StatefulSet and
// returns its readiness
func CheckStatefulSetIsReady(ctx context.Context, cli appsv1client.StatefulSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		s, err := cli.Get(ctx, name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return StatefulSetIsReady(s), nil
	}
}

// MachineConfigPoolIsReady returns true if a MachineConfigPool is considered
// ready
func MachineConfigPoolIsReady(s *mcv1.MachineConfigPool) bool {
	return s.Status.MachineCount == s.Status.UpdatedMachineCount &&
		s.Status.MachineCount == s.Status.ReadyMachineCount &&
		s.Generation == s.Status.ObservedGeneration
}

// CheckMachineConfigPoolIsReady returns a function which polls a
// MachineConfigPool and returns its readiness
func CheckMachineConfigPoolIsReady(ctx context.Context, cli machineconfigurationv1.MachineConfigPoolInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		s, err := cli.Get(ctx, name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return MachineConfigPoolIsReady(s), nil
	}
}
