package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func SetCondition(ctx context.Context, arocli aroclient.Interface, cond *status.Condition, role string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		changed := cluster.Status.Conditions.SetCondition(*cond)

		if role == operator.RoleMaster && cluster.Status.OperatorVersion != version.GitCommit {
			cluster.Status.OperatorVersion = version.GitCommit
			changed = true
		}

		if !changed {
			return nil
		}

		_, err = arocli.AroV1alpha1().Clusters().UpdateStatus(ctx, cluster, metav1.UpdateOptions{})
		return err
	})
}
