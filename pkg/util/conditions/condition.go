package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// clock is used to set status condition timestamps.
// This variable makes it easier to test conditions.
var kubeclock clock.Clock = &clock.RealClock{}

func SetCondition(ctx context.Context, c client.Client, cond *operatorv1.OperatorCondition, role string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if cond == nil {
			return nil
		}

		cluster := &arov1alpha1.Cluster{}
		err := c.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
		if err != nil {
			return err
		}

		var changed bool
		cluster.Status.Conditions, changed = setCondition(cluster.Status.Conditions, *cond)

		if cleanStaleConditions(cluster, role) {
			changed = true
		}

		if !changed {
			return nil
		}

		return c.Status().Update(ctx, cluster)
	})
}

func IsTrue(conditions []operatorv1.OperatorCondition, t string) bool {
	return isCondition(conditions, t, operatorv1.ConditionTrue)
}

func IsFalse(conditions []operatorv1.OperatorCondition, t string) bool {
	return isCondition(conditions, t, operatorv1.ConditionFalse)
}

func isCondition(conditions []operatorv1.OperatorCondition, t string, s operatorv1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == t && condition.Status == s {
			return true
		}
	}
	return false
}

// cleanStaleConditions ensures that conditions no longer in use as defined by older operators
// are always removed from the cluster.status.conditions
func cleanStaleConditions(cluster *arov1alpha1.Cluster, role string) (changed bool) {
	conditions := make([]operatorv1.OperatorCondition, 0, len(cluster.Status.Conditions))

	// cleanup any old conditions
	current := map[string]bool{}
	for _, ct := range arov1alpha1.AllConditionTypes() {
		current[ct] = true
	}

	for _, cond := range cluster.Status.Conditions {
		if _, ok := current[cond.Type]; ok || conditionIsControllerStatus(cond.Type) {
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

func conditionIsControllerStatus(conditionType string) bool {
	return strings.HasSuffix(conditionType, "Controller"+operatorv1.OperatorStatusTypeAvailable) ||
		strings.HasSuffix(conditionType, "Controller"+operatorv1.OperatorStatusTypeProgressing) ||
		strings.HasSuffix(conditionType, "Controller"+operatorv1.OperatorStatusTypeDegraded)
}

// SetCondition adds (or updates) the set of conditions with the given
// condition. It returns a boolean value indicating whether the set condition
// is new or was a change to the existing condition with the same type.
func setCondition(conditions []operatorv1.OperatorCondition, newCond operatorv1.OperatorCondition) ([]operatorv1.OperatorCondition, bool) {
	newCond.LastTransitionTime = metav1.Time{Time: kubeclock.Now()}

	for i, condition := range conditions {
		if condition.Type == newCond.Type {
			if condition.Status == newCond.Status {
				newCond.LastTransitionTime = condition.LastTransitionTime
			}
			changed := condition.Status != newCond.Status ||
				condition.Reason != newCond.Reason ||
				condition.Message != newCond.Message
			(conditions)[i] = newCond
			return conditions, changed
		}
	}
	conditions = append(conditions, newCond)
	return conditions, true
}
