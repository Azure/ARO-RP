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

		if cleanStaleConditions(cluster, role) {
			changed = true
		}

		if !changed {
			return nil
		}

		_, err = arocli.AroV1alpha1().Clusters().UpdateStatus(ctx, cluster, metav1.UpdateOptions{})
		return err
	})
}

// cleanStaleConditions ensures that conditions no longer in use as defined by older operators
// are always removed from the cluster.status.conditions
func cleanStaleConditions(cluster *arov1alpha1.Cluster, role string) (changed bool) {
	conditions := make(status.Conditions, 0, len(cluster.Status.Conditions))

	// cleanup any old conditions
	current := map[status.ConditionType]bool{}
	for _, ct := range arov1alpha1.AllConditionTypes() {
		current[ct] = true
	}

	for _, cond := range cluster.Status.Conditions {
		if _, ok := current[cond.Type]; ok {
			conditions = append(conditions, cond)
		} else {
			changed = true
		}
	}

	cluster.Status.Conditions = conditions

	if role == operator.RoleMaster && cluster.Status.OperatorVersion != version.GitCommit {
		cluster.Status.OperatorVersion = version.GitCommit
		changed = true
	}

	return
}
