package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclientv1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
		ds.Status.CurrentNumberScheduled == ds.Status.NumberReady &&
		ds.Status.CurrentNumberScheduled > 0 &&
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
		case kerrors.IsNotFound(err):
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
		case kerrors.IsNotFound(err):
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
		case kerrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return PodIsRunning(p), nil
	}
}

// CheckPodsAreRunning returns a function which polls multiple Pods by label and returns if it is
// running
func CheckPodsAreRunning(ctx context.Context, cli corev1client.PodInterface, labels map[string]string) func() (bool, error) {
	return func() (bool, error) {
		// build the label selector
		options := metav1.ListOptions{
			LabelSelector: "",
		}
		for key, value := range labels {
			options.LabelSelector += key + "=" + value
		}

		// get list of pods
		podList, err := cli.List(ctx, options)

		// check status from List
		switch {
		case kerrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		// Test if each pod is running
		// 	return true if all pods are running
		// 	else return false
		for _, p := range podList.Items {
			if !PodIsRunning(&p) {
				return false, nil
			}
		}

		// didn't fail any tests, return true
		return true, nil
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

// MachineConfigPoolIsReady returns true if a MachineConfigPool is considered
// ready
func MachineConfigPoolIsReady(s *mcv1.MachineConfigPool) bool {
	return s.Status.MachineCount == s.Status.UpdatedMachineCount &&
		s.Status.MachineCount == s.Status.ReadyMachineCount &&
		s.Generation == s.Status.ObservedGeneration
}

// CheckMachineConfigPoolIsReady returns a function which polls a
// MachineConfigPool and returns its readiness
func CheckMachineConfigPoolIsReady(ctx context.Context, cli mcoclientv1.MachineConfigPoolInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		s, err := cli.Get(ctx, name, metav1.GetOptions{})
		switch {
		case kerrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return MachineConfigPoolIsReady(s), nil
	}
}

type MCPLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*mcv1.MachineConfigPoolList, error)
}

type NodeLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.NodeList, error)
}

// SameNumberOfNodesAndMachines returns true if the cluster has the same number of nodes and total machines,
// and an error if any. Otherwise it returns false and nil.
func SameNumberOfNodesAndMachines(ctx context.Context, mcpLister MCPLister, nodeLister NodeLister) (bool, error) {
	if mcpLister == nil {
		return false, fmt.Errorf("mcpLister is nil")
	}

	if nodeLister == nil {
		return false, fmt.Errorf("nodeLister is nil")
	}

	machineConfigPoolList, err := mcpLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	totalMachines, err := TotalMachinesInTheMCPs(machineConfigPoolList.Items)
	if err != nil {
		return false, err
	}

	nodes, err := nodeLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	nNodes := len(nodes.Items)
	if nNodes != totalMachines {
		return false, fmt.Errorf("cluster has %d nodes but %d under MCPs, not removing private DNS zone", nNodes, totalMachines)
	}

	return true, nil
}

// TotalMachinesInTheMCPs returns the total number of machines in the machineConfigPools
// and an error, if any.
func TotalMachinesInTheMCPs(machineConfigPools []mcv1.MachineConfigPool) (int, error) {
	totalMachines := 0
	for _, mcp := range machineConfigPools {
		if !MCPContainsARODNSConfig(mcp) {
			return 0, fmt.Errorf("ARO DNS config not found in MCP %s", mcp.Name)
		}

		if !MachineConfigPoolIsReady(&mcp) {
			return 0, fmt.Errorf("MCP %s not ready", mcp.Name)
		}

		totalMachines += int(mcp.Status.MachineCount)
	}
	return totalMachines, nil
}

func MCPContainsARODNSConfig(mcp mcv1.MachineConfigPool) bool {
	for _, source := range mcp.Status.Configuration.Source {
		mcpPrefix := "99-"
		mcpSuffix := "-aro-dns"

		if source.Name == mcpPrefix+mcp.Name+mcpSuffix {
			return true
		}
	}
	return false
}
