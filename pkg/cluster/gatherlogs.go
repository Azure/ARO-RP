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
)

type diagnosticStep struct {
	f func(context.Context) error
}

func (m *manager) gatherFailureLogs(ctx context.Context, runType string) {
	d := failurediagnostics.NewFailureDiagnostics(
		m.log,
		m.env,
		m.doc,
		m.virtualMachines,
		m.loadBalancers,
		m.metrics,
	)

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
		err := f.f(ctx)
		if err != nil {
			m.log.Errorf("failed to gather logs: %v", err)
			continue
		}
	}
}

func (m *manager) logClusterDeployment(ctx context.Context) error {
	if m.doc == nil || m.hiveClusterManager == nil {
		m.log.Info("skipping step")
		return nil
	}

	cd, err := m.hiveClusterManager.GetClusterDeployment(ctx, m.doc)
	if err != nil {
		m.log.WithError(err).Errorf("failed to get cluster deployment")
		return err
	}

	cd.ManagedFields = nil
	m.log.Infof("clusterdeployment %s - %s", cd.Name, structToJson(cd))

	return nil
}

func (m *manager) logClusterVersion(ctx context.Context) error {
	if m.configcli == nil {
		m.log.Info("skipping step")
		return nil
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		m.log.WithError(err).Errorf("failed to get clusterversion")
		return err
	}

	cv.ManagedFields = nil
	m.log.Infof("clusterversion %s - %s", cv.Name, structToJson(cv))

	return nil
}

func (m *manager) logNodes(ctx context.Context) error {
	if m.kubernetescli == nil {
		m.log.Info("skipping step")
		return nil
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.WithError(err).Errorf("failed to get nodes")
		return err
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("no nodes found")
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

		m.log.Infof("node %s - Ready: %s", node.Name, nodeReady)
		m.log.Infof("node %s - %s", node.Name, structToJson(node))
	}

	return nil
}

func (m *manager) logClusterOperators(ctx context.Context) error {
	if m.configcli == nil {
		m.log.Info("skipping step")
		return nil
	}

	cos, err := m.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.WithError(err).Errorf("failed to get clusteroperators")
		return err
	}

	if len(cos.Items) == 0 {
		return fmt.Errorf("no cluster operators found")
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
		m.log.Infof("clusteroperator %s - Available: %s, Progressing: %s, Degraded: %s", co.Name, coAvailable, coProgressing, coDegraded)
		m.log.Infof("clusteroperator %s - %s", co.Name, structToJson(co))
	}

	return nil
}

func (m *manager) logIngressControllers(ctx context.Context) error {
	if m.operatorcli == nil {
		m.log.Info("skipping step")
		return nil
	}

	ics, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.WithError(err).Errorf("failed to get ingresscontrollers")
		return err
	}

	if len(ics.Items) == 0 {
		return fmt.Errorf("no ingress controllers found")
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
		m.log.Infof("ingresscontroller %s - Available: %s, Progressing: %s, Degraded: %s", ic.Name, icAvailable, icProgressing, icDegraded)
		m.log.Infof("ingresscontroller %s - %s", ic.Name, structToJson(ic))
	}

	return nil
}

func (m *manager) logPodLogs(ctx context.Context) error {
	if m.kubernetescli == nil {
		m.log.Info("skipping step")
		return nil
	}

	tailLines := int64(20)
	podLogOptions := corev1.PodLogOptions{
		TailLines: &tailLines,
	}

	for _, ns := range []string{"openshift-azure-operator", "openshift-machine-config-operator"} {
		pods, err := m.kubernetescli.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			m.log.WithError(err).Errorf("failed to list pods in namespace %s", ns)
			return err
		}
		for _, i := range pods.Items {
			podName := fmt.Sprintf("%s/%s", i.Namespace, i.Name)
			l := m.log.WithField("pod", podName)
			l.Infof("pod %s - phase=%s reason=%s message=%s", podName, i.Status.Phase, i.Status.Reason, i.Status.Message)
			for _, condition := range i.Status.Conditions {
				l.Infof("pod %s - Condition %s=%s reason=%s transition=%s message=%s", podName, condition.Type, condition.Status, condition.Reason, condition.LastTransitionTime, condition.Message)
			}
			for _, containerStatus := range i.Status.ContainerStatuses {
				l.Infof("pod %s - Container %s started=%t ready=%t restarts=%d state=%s", podName, containerStatus.Name, *containerStatus.Started, containerStatus.Ready, containerStatus.RestartCount, structToJson(containerStatus.State))
			}

			req := m.kubernetescli.CoreV1().Pods(ns).GetLogs(i.Name, &podLogOptions)
			logStream, err := req.Stream(ctx)
			if err != nil {
				m.log.Infof("pod logs retrieval error for %s: %s", i.Name, err)
				continue
			}
			defer logStream.Close()

			reader := bufio.NewReader(logStream)
			for {
				line, err := reader.ReadString('\n')
				l.Info(strings.TrimSpace(line))
				if err == io.EOF {
					break
				}
				if err != nil {
					l.Infof("pod logs reading error for %s/%s", i.Namespace, i.Name)
					break
				}
			}
		}
	}
	return nil
}

func structToJson(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	// Replace double quotes with single quotes
	return strings.ReplaceAll(string(b), "\"", "'")
}
