package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/cluster/failurediagnostics"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

type diagnosticStep struct {
	f func(context.Context) ([]string, error)
}

func (m *manager) gatherFailureLogs(ctx context.Context, runType string) {
	d := failurediagnostics.NewFailureDiagnostics(m.log, m.env, m.doc, m.virtualMachines, m.loadBalancers)

	s := []diagnosticStep{
		{f: m.logClusterVersion},
		{f: m.logNodes},
		{f: m.logClusterOperators},
		{f: m.logIngressControllers},
		{f: m.logPodLogs},
	}

	// only log serial consoles and Hive CD on an install, not on updates/adminUpdates
	if runType == "install" {
		s = append(s, diagnosticStep{f: d.LogVMSerialConsole})
		s = append(s, diagnosticStep{f: d.LogLoadBalancers})
		s = append(s, diagnosticStep{f: m.logClusterDeployment})
	}

	for _, f := range s {
		log := m.log.WithField("func", steps.FriendlyName(f.f))
		s, err := f.f(ctx)
		for _, i := range s {
			log.Info(i)
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func (m *manager) logClusterDeployment(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.doc == nil || m.hiveClusterManager == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	cd, err := m.hiveClusterManager.GetClusterDeployment(ctx, m.doc)
	if err != nil {
		return lines, err
	}

	cd.ManagedFields = nil
	lines = append(lines, fmt.Sprintf("clusterdeployment %s - %s", cd.Name, structToJson(cd)))

	return lines, nil
}

func (m *manager) logClusterVersion(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.configcli == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return lines, err
	}

	cv.ManagedFields = nil
	lines = append(lines, fmt.Sprintf("clusterversion %s - %s", cv.Name, structToJson(cv)))

	return lines, nil
}

func (m *manager) logNodes(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.kubernetescli == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return lines, err
	}

	if len(nodes.Items) == 0 {
		return lines, fmt.Errorf("no nodes found")
	}

	for _, node := range nodes.Items {
		node.ManagedFields = nil

		nodeReady := corev1.ConditionUnknown
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				nodeReady = condition.Status
				break
			}
		}

		lines = append(lines, fmt.Sprintf("node %s - Ready: %s", node.Name, nodeReady))
		lines = append(lines, fmt.Sprintf("node %s - %s", node.Name, structToJson(node)))
	}

	return lines, nil
}

func (m *manager) logClusterOperators(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.configcli == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	cos, err := m.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return lines, err
	}

	if len(cos.Items) == 0 {
		return lines, fmt.Errorf("no cluster operators found")
	}

	for _, co := range cos.Items {
		co.ManagedFields = nil

		coAvailable := configv1.ConditionUnknown
		coProgressing := configv1.ConditionUnknown
		coDegraded := configv1.ConditionUnknown
		for _, condition := range co.Status.Conditions {
			switch condition.Type {
			case configv1.OperatorAvailable:
				coAvailable = condition.Status
			case configv1.OperatorProgressing:
				coProgressing = condition.Status
			case configv1.OperatorDegraded:
				coDegraded = condition.Status
			}
		}
		lines = append(lines, fmt.Sprintf("clusteroperator %s - Available: %s, Progressing: %s, Degraded: %s", co.Name, coAvailable, coProgressing, coDegraded))
		lines = append(lines, fmt.Sprintf("clusteroperator %s - %s", co.Name, structToJson(co)))
	}

	return lines, nil
}

func (m *manager) logIngressControllers(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.operatorcli == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	ics, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return lines, err
	}

	if len(ics.Items) == 0 {
		return lines, fmt.Errorf("no ingress controllers found")
	}

	for _, ic := range ics.Items {
		ic.ManagedFields = nil

		icAvailable := operatorv1.ConditionUnknown
		icProgressing := operatorv1.ConditionUnknown
		icDegraded := operatorv1.ConditionUnknown
		for _, condition := range ic.Status.Conditions {
			switch condition.Type {
			case operatorv1.OperatorStatusTypeAvailable:
				icAvailable = condition.Status
			case operatorv1.OperatorStatusTypeProgressing:
				icProgressing = condition.Status
			case operatorv1.OperatorStatusTypeDegraded:
				icDegraded = condition.Status
			}
		}
		lines = append(lines, fmt.Sprintf("ingresscontroller %s - Available: %s, Progressing: %s, Degraded: %s", ic.Name, icAvailable, icProgressing, icDegraded))
		lines = append(lines, fmt.Sprintf("ingresscontroller %s - %s", ic.Name, structToJson(ic)))
	}

	return lines, nil
}

func (m *manager) logPodLogs(ctx context.Context) ([]string, error) {
	lines := make([]string, 0)

	if m.kubernetescli == nil {
		lines = append(lines, "skipping step")
		return lines, nil
	}

	tailLines := int64(20)
	podLogOptions := corev1.PodLogOptions{
		TailLines: &tailLines,
	}

	for _, ns := range []string{"openshift-azure-operator", "openshift-machine-config-operator"} {
		pods, err := m.kubernetescli.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, i := range pods.Items {
			lines = append(lines, formatPodStatus(i)...)

			req := m.kubernetescli.CoreV1().Pods(ns).GetLogs(i.Name, &podLogOptions)
			logStream, err := req.Stream(ctx)
			if err != nil {
				lines = append(lines, fmt.Sprintf("pod logs retrieval error for %s: %s", i.Name, err))
				continue
			}
			defer logStream.Close()

			reader := bufio.NewReader(logStream)
			for {
				line, err := reader.ReadString('\n')
				lines = append(lines, fmt.Sprintf("pod %s/%s | %s", i.Namespace, i.Name, strings.TrimSpace(line)))
				if err == io.EOF {
					break
				}
				if err != nil {
					lines = append(lines, fmt.Sprintf("pod logs reading error for %s/%s", i.Namespace, i.Name))
					break
				}
			}
		}
	}
	return lines, nil
}

func structToJson(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	// Replace double quotes with single quotes
	return strings.ReplaceAll(string(b), "\"", "'")
}

func formatPodStatus(pod corev1.Pod) []string {
	items := make([]string, 0)
	prefix := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	items = append(items, fmt.Sprintf("pod %s - phase=%s reason=%s message=%s", prefix, pod.Status.Phase, pod.Status.Reason, pod.Status.Message))
	for _, condition := range pod.Status.Conditions {
		items = append(items, fmt.Sprintf("pod %s - Condition %s=%s reason=%s transition=%s message=%s", prefix, condition.Type, condition.Status, condition.Reason, condition.LastTransitionTime, condition.Message))
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		items = append(items, fmt.Sprintf("pod %s - Container %s started=%t ready=%t restarts=%d state=%s", prefix, containerStatus.Name, *containerStatus.Started, containerStatus.Ready, containerStatus.RestartCount, structToJson(containerStatus.State)))
	}
	return items
}
