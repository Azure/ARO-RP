package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/clusteroperators"
)

const minimumWorkerNodes = 2
const workerMachineRoleLabel = "machine.openshift.io/cluster-api-machine-role=worker"
const workerNodeRoleLabel = "node-role.kubernetes.io/worker"
const phaseRunning = "Running"

var clusterOperatorsToRequireSettled = []string{"kube-controller-manager", "kube-apiserver", "kube-scheduler", "console", "authentication"}

// condition functions should return an error only if it's not able to be retried
// if a condition function encounters a error when retrying it should return false, nil.

func (m *manager) apiServersReady(ctx context.Context) (bool, error) {
	apiserver, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "kube-apiserver", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return clusteroperators.IsOperatorAvailable(apiserver), nil
}

func (m *manager) minimumWorkerNodesReady(ctx context.Context) (bool, error) {
	machines, err := m.maocli.MachineV1beta1().Machines("openshift-machine-api").List(ctx, metav1.ListOptions{
		LabelSelector: workerMachineRoleLabel,
	})
	if err != nil {
		m.log.Error(err)
		return false, nil
	}

	readyWorkerMachines := 0
	for _, machine := range machines.Items {
		if machine.Status.Phase == nil || machine.Status.ProviderStatus == nil {
			m.log.Infof("Unable to determine status of machine %s: %v", machine.Name, machine.Status)
			break
		}
		m.log.Infof("Machine %s is %s; status: %s", machine.Name, *machine.Status.Phase, string(machine.Status.ProviderStatus.Raw))
		if *machine.Status.Phase == phaseRunning {
			readyWorkerMachines++
		}
	}

	if readyWorkerMachines < minimumWorkerNodes {
		m.log.Infof("%d machines running, need at least %d", readyWorkerMachines, minimumWorkerNodes)
		return false, nil
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: workerNodeRoleLabel,
	})
	if err != nil {
		return false, nil
	}

	readyWorkers := 0
	for _, node := range nodes.Items {
		m.log.Infof("Node %s status: %v", node.Name, node.Status.Conditions)
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				readyWorkers++
			}
		}
	}

	m.log.Infof("%d nodes ready, need at least %d", readyWorkerMachines, minimumWorkerNodes)
	return readyWorkers >= minimumWorkerNodes, nil
}

func (m *manager) operatorConsoleExists(ctx context.Context) (bool, error) {
	_, err := m.operatorcli.OperatorV1().Consoles().Get(ctx, consoleConfigResourceName, metav1.GetOptions{})
	return err == nil, nil
}

func (m *manager) operatorConsoleReady(ctx context.Context) (bool, error) {
	consoleOperator, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "console", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return clusteroperators.IsOperatorAvailable(consoleOperator), nil
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
	return clusteroperators.IsOperatorAvailable(ingressOperator), nil
}

// aroCredentialsRequestReconciled evaluates whether the openshift-azure-operator CredentialsRequest has recently been reconciled and returns true
// if it has been (or does not need to be under the circumstances) and false otherwise or if an error occurs, where "has recently been reconciled"\
// is true if the CredentialsRequest has been reconciled within the past 5 minutes.
// Checking for a change to the lastSyncCloudCredsSecretResourceVersion attribute of the CredentialRequest's status would be a neater way of checking
// whether it was reconciled, but we would would have to save the value prior to updating the kube-system/azure-credentials Secret so that we'd have
// an old value to compare to.
func (m *manager) aroCredentialsRequestReconciled(ctx context.Context) (bool, error) {
	// If the CSP hasn't been updated, the CredentialsRequest does not need to be reconciled.
	secret, err := m.servicePrincipalUpdated(ctx)
	if err != nil {
		return false, err
	} else if secret == nil {
		return true, nil
	}

	u, err := m.dynamiccli.Resource(CredentialsRequestGroupVersionResource).Namespace("openshift-cloud-credential-operator").Get(ctx, "openshift-azure-operator", metav1.GetOptions{})
	if err != nil {
		// If the CredentialsRequest is not found, it may have just recently been reconciled.
		// Return nil to retry until we hit the condition timeout.
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	cr := u.UnstructuredContent()
	var status map[string]interface{}
	if s, ok := cr["status"]; ok {
		status = s.(map[string]interface{})
	} else {
		return false, errors.New("unable to access status of openshift-azure-operator CredentialsRequest")
	}

	var lastSyncTimestamp string
	if lst, ok := status["lastSyncTimestamp"]; ok {
		lastSyncTimestamp = lst.(string)
	} else {
		return false, errors.New("unable to access status.lastSyncTimestamp of openshift-azure-operator CredentialsRequest")
	}

	timestamp, err := time.Parse(time.RFC3339, lastSyncTimestamp)
	if err != nil {
		return false, err
	}

	timeSinceLastSync := time.Since(timestamp)
	return timeSinceLastSync.Minutes() < 5, nil
}

// Check if all ClusterOperators have settled (i.e. are available and not
// progressing).
func (m *manager) clusterOperatorsHaveSettled(ctx context.Context) (bool, error) {
	coList := &configv1.ClusterOperatorList{}

	err := m.kubeClientHelper.List(ctx, coList)
	if err != nil {
		// Be resilient to failures as kube-apiserver might drop connections while it's reconciling
		m.log.Errorf("failure listing cluster operators, retrying: %s", err.Error())
		return false, nil
	}

	allSettled := true

	// Only check the COs we care about to prevent added ones in new OpenShift
	// versions perhaps tripping us up later
	for _, co := range coList.Items {
		if slices.Contains(clusterOperatorsToRequireSettled, strings.ToLower(co.Name)) {
			if !clusteroperators.IsOperatorAvailable(&co) {
				allSettled = false
				m.log.Warnf("ClusterOperator not yet settled: %s", clusteroperators.OperatorStatusText(&co))
			}
		}
	}

	return allSettled, nil
}
