package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

const minimumWorkerNodes = 2

// condition functions should return an error only if it's not able to be retried
// if a condition function encounters a error when retrying it should return false, nil.

func (m *manager) apiServersReady(ctx context.Context) (bool, retry bool, err error) {
	apiserver, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "kube-apiserver", metav1.GetOptions{})
	if err != nil {
		return false, true, nil
	}
	return isOperatorAvailable(apiserver), true, nil
}

func getErrMessage(err error, messageifany string) error {
	message := "Minimum number of worker nodes have not been successfully created. Please retry and if the issue persists, raise an Azure support ticket"
	if err != nil {
		message = "Error: " + err.Error() + "Message: " + messageifany + message
	} else {
		message = messageifany + message
	}
	cloudError := api.NewCloudError(
		http.StatusInternalServerError,
		api.CloudErrorCodeDeploymentFailed,
		"",
		message,
	)
	return cloudError
}

func (m *manager) minimumWorkerNodesReady(ctx context.Context) (nodeCheck bool, retry bool, err error) {
	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return false, true, getErrMessage(err, "")
	}

	readyWorkers := 0
	message := ""
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				readyWorkers++
			} else {
				messageString := fmt.Sprintf("%+v - Status:%+v, Message: %+v\n", node, cond.Status, cond.Message)
				message += messageString
			}
		}
	}
	minWorkerAchieved := readyWorkers >= minimumWorkerNodes
	if minWorkerAchieved {
		return minWorkerAchieved, false, nil
	} else {
		if message == "" {
			message = "Check the config and versions"
		}
		return false, true, getErrMessage(err, message)
	}
}

func (m *manager) operatorConsoleExists(ctx context.Context) (errorcheck bool, retry bool, err error) {
	_, err = m.operatorcli.OperatorV1().Consoles().Get(ctx, consoleConfigResourceName, metav1.GetOptions{})
	return err == nil, false, nil
}

func (m *manager) operatorConsoleReady(ctx context.Context) (consoleOperatorcheck bool, retry bool, err error) {
	consoleOperator, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "console", metav1.GetOptions{})
	if err != nil {
		return false, true, nil
	}
	return isOperatorAvailable(consoleOperator), true, nil
}

func (m *manager) clusterVersionReady(ctx context.Context) (cvcheck bool, retry bool, err error) {
	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err == nil {
		for _, cond := range cv.Status.Conditions {
			if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
				return true, false, nil
			}
		}
	}
	return false, true, nil
}

func (m *manager) ingressControllerReady(ctx context.Context) (ingressOperatorcheck bool, retry bool, err error) {
	ingressOperator, err := m.configcli.ConfigV1().ClusterOperators().Get(ctx, "ingress", metav1.GetOptions{})
	if err != nil {
		return false, true, nil
	}
	ingressOperatorcheck = isOperatorAvailable(ingressOperator)
	if ingressOperatorcheck {
		return ingressOperatorcheck, false, nil
	} else {
		return ingressOperatorcheck, true, nil
	}
}

func isOperatorAvailable(operator *configv1.ClusterOperator) bool {
	m := make(map[configv1.ClusterStatusConditionType]configv1.ConditionStatus, len(operator.Status.Conditions))
	for _, cond := range operator.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[configv1.OperatorAvailable] == configv1.ConditionTrue && m[configv1.OperatorProgressing] == configv1.ConditionFalse
}

// aroCredentialsRequestReconciled evaluates whether the openshift-azure-operator CredentialsRequest has recently been reconciled and returns true
// if it has been (or does not need to be under the circumstances) and false otherwise or if an error occurs, where "has recently been reconciled"\
// is true if the CredentialsRequest has been reconciled within the past 5 minutes.
// Checking for a change to the lastSyncCloudCredsSecretResourceVersion attribute of the CredentialRequest's status would be a neater way of checking
// whether it was reconciled, but we would would have to save the value prior to updating the kube-system/azure-credentials Secret so that we'd have
// an old value to compare to.
func (m *manager) aroCredentialsRequestReconciled(ctx context.Context) (credcheck bool, retry bool, err error) {
	// If the CSP hasn't been updated, the CredentialsRequest does not need to be reconciled.
	secret, err := m.servicePrincipalUpdated(ctx)
	if err != nil {
		return false, false, err
	} else if secret == nil {
		return true, false, nil
	}

	u, err := m.dynamiccli.Resource(CredentialsRequestGroupVersionResource).Namespace("openshift-cloud-credential-operator").Get(ctx, "openshift-azure-operator", metav1.GetOptions{})
	if err != nil {
		// If the CredentialsRequest is not found, it may have just recently been reconciled.
		// Return nil to retry until we hit the condition timeout.
		if kerrors.IsNotFound(err) {
			return false, true, nil
		}
		return false, false, err
	}

	cr := u.UnstructuredContent()
	var status map[string]interface{}
	if s, ok := cr["status"]; ok {
		status = s.(map[string]interface{})
	} else {
		return false, false, errors.New("unable to access status of openshift-azure-operator CredentialsRequest")
	}

	var lastSyncTimestamp string
	if lst, ok := status["lastSyncTimestamp"]; ok {
		lastSyncTimestamp = lst.(string)
	} else {
		return false, false, errors.New("unable to access status.lastSyncTimestamp of openshift-azure-operator CredentialsRequest")
	}

	timestamp, err := time.Parse(time.RFC3339, lastSyncTimestamp)
	if err != nil {
		return false, false, err
	}

	timeSinceLastSync := time.Since(timestamp)
	timeSinceLastSyncCheck := timeSinceLastSync.Minutes() < 5
	if timeSinceLastSyncCheck {
		return true, false, nil
	} else {
		return false, true, nil
	}
}
