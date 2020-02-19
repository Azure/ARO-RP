package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (i *Installer) waitForClusterVersion(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err == nil {
			for _, cond := range cv.Status.Conditions {
				if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil

	}, timeoutCtx.Done())
}

func (i *Installer) waitForBootstrapConfigmap(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		cm, err := i.kubernetescli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
		return err == nil && cm.Data["status"] == "complete", nil

	}, timeoutCtx.Done())
}
