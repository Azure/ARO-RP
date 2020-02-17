package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func (i *Installer) updateConsoleBranding(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	cli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.log.Print("waiting for console-operator config")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		_, err := cli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		return err == nil, nil

	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	i.log.Print("updating console-operator branding")
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		operatorConfig, err := cli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		operatorConfig.Spec.Customization.Brand = operatorv1.BrandAzure

		_, err = cli.OperatorV1().Consoles().Update(operatorConfig)
		return err
	})
	if err != nil {
		return err
	}

	i.log.Print("waiting for console to reload")
	timeoutCtx, cancel = context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		operatorConfig, err := cli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		if err == nil && operatorConfig.Status.ObservedGeneration == operatorConfig.Generation {
			for _, cond := range operatorConfig.Status.Conditions {
				if cond.Type == "Deployment"+operatorv1.OperatorStatusTypeAvailable &&
					cond.Status == operatorv1.ConditionTrue {
					return true, nil
				}
			}
		}

		return false, nil
	}, timeoutCtx.Done())
}
