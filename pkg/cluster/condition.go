package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	configv1 "github.com/openshift/api/config/v1"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const minimumWorkerNodes = 2

// condition functions should return an error only if it's not retryable
// if a condition function encounters a retryable error it should return false, nil.

func (m *manager) bootstrapConfigMapReady(ctx context.Context) (bool, error) {
	cm, err := m.kubernetescli.CoreV1().ConfigMaps("kube-system").Get(ctx, "bootstrap", metav1.GetOptions{})
	if err != nil && m.env.IsLocalDevelopmentMode() {
		m.log.Printf("bootstrapConfigMapReady condition error %s", err)
	}
	m.log.Printf(spew.Sdump(cm))
	return err == nil && cm.Data["status"] == "complete", nil
}

func (m *manager) apiServersReady(ctx context.Context) (bool, error) {
	apiserver, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "kube-apiserver", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return isOperatorAvailable(apiserver), nil
}

func (m *manager) minimumWorkerNodesReady(ctx context.Context) (bool, error) {
	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return false, nil
	}

	readyWorkers := 0
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				readyWorkers++
			}
		}
	}

	return readyWorkers >= minimumWorkerNodes, nil
}

func (m *manager) operatorConsoleExists(ctx context.Context) (bool, error) {
	_, err := m.operatorcli.OperatorV1().Consoles().Get(ctx, consoleapi.ConfigResourceName, metav1.GetOptions{})
	return err == nil, nil
}

func (m *manager) operatorConsoleReady(ctx context.Context) (bool, error) {
	consoleOperator, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "console", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return isOperatorAvailable(consoleOperator), nil
}

func (m *manager) clusterVersionReady(ctx context.Context) (bool, error) {
	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err == nil {
		for _, cond := range cv.Status.Conditions {
			if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *manager) ingressControllerReady(ctx context.Context) (bool, error) {
	ingressOperator, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "ingress", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return isOperatorAvailable(ingressOperator), nil
}

func isOperatorAvailable(operator *configv1.ClusterOperator) bool {
	m := make(map[configv1.ClusterStatusConditionType]configv1.ConditionStatus, len(operator.Status.Conditions))
	for _, cond := range operator.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[configv1.OperatorAvailable] == configv1.ConditionTrue && m[configv1.OperatorProgressing] == configv1.ConditionFalse
}
