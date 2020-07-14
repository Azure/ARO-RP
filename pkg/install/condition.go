package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// condition functions should return an error only if it's not retryable
// if a condition function encounters a retryable error it should return false, nil.

func (i *Installer) bootstrapConfigMapReady(ctx context.Context) (bool, error) {
	cm, err := i.kubernetescli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
	return err == nil && cm.Data["status"] == "complete", nil
}

func (i *Installer) apiServersReady(ctx context.Context) (bool, error) {
	apiserver, err := i.configcli.ConfigV1().ClusterOperators().Get("kube-apiserver", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return isOperatorAvailable(apiserver), nil
}

func (i *Installer) operatorConsoleExists(ctx context.Context) (bool, error) {
	_, err := i.operatorcli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	return err == nil, nil
}

func (i *Installer) operatorConsoleReady(ctx context.Context) (bool, error) {
	consoleOperator, err := i.configcli.ConfigV1().ClusterOperators().Get("console", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return isOperatorAvailable(consoleOperator), nil
}

func (i *Installer) clusterVersionReady(ctx context.Context) (bool, error) {
	cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err == nil {
		for _, cond := range cv.Status.Conditions {
			if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

func (i *Installer) ingressControllerReady(ctx context.Context) (bool, error) {
	ingressOperator, err := i.configcli.ConfigV1().ClusterOperators().Get("ingress", metav1.GetOptions{})
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
