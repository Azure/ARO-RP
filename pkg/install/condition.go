package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/env"
)

func (i *Installer) bootstrapConfigMapReady() (bool, error) {
	cm, err := i.kubernetescli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
	return err == nil && cm.Data["status"] == "complete", nil
}

func (i *Installer) apiServersReady() (bool, error) {
	apiserver, err := i.operatorcli.OperatorV1().KubeAPIServers().Get("cluster", metav1.GetOptions{})
	if err == nil {
		m := make(map[string]operatorv1.ConditionStatus, len(apiserver.Status.Conditions))
		for _, cond := range apiserver.Status.Conditions {
			m[cond.Type] = cond.Status
		}
		if m["Available"] == operatorv1.ConditionTrue && m["Progressing"] == operatorv1.ConditionFalse {
			return true, nil
		}
	}
	return false, nil
}

func (i *Installer) operatorConsoleExists() (bool, error) {
	_, err := i.operatorcli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	return err == nil, nil
}

func (i *Installer) operatorConsoleReady() (bool, error) {
	operatorConfig, err := i.operatorcli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err == nil && operatorConfig.Status.ObservedGeneration == operatorConfig.Generation {
		for _, cond := range operatorConfig.Status.Conditions {
			if cond.Type == "Deployment"+operatorv1.OperatorStatusTypeAvailable &&
				cond.Status == operatorv1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

func (i *Installer) clusterVersionReady() (bool, error) {
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

func (i *Installer) ingressControllerReady() (bool, error) {
	ic, err := i.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get("default", metav1.GetOptions{})
	if err == nil && ic.Status.ObservedGeneration == ic.Generation {
		if _, ok := i.env.(env.Dev); !ok {
			if ic.Spec.DefaultCertificate == nil {
				return false, nil
			}
		}
		for _, cond := range ic.Status.Conditions {
			if cond.Type == operatorv1.OperatorStatusTypeAvailable && cond.Status == operatorv1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}
