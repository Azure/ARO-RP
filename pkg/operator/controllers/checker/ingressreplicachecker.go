// Implements a check that provides detail on potentially faulty or customised
// IngressController replica spec on the default controller.
//
// Included checks are:
//  - existence of default ingresscontroller
//  - if the ingresscontroller replica is downgraded to 0

package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

type IngressReplicaChecker struct {
	arocli      aroclient.Interface
	operatorcli operatorclient.Interface
	role        string
}

func NewIngressReplicaChecker(arocli aroclient.Interface, operatorcli operatorclient.Interface, role string) *IngressReplicaChecker {
	return &IngressReplicaChecker{
		arocli:      arocli,
		operatorcli: operatorcli,
		role:        role,
	}
}

func (r *IngressReplicaChecker) Name() string {
	return "IngressReplicaChecker"
}

func (r *IngressReplicaChecker) Check(ctx context.Context) error {
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.DefaultIngressReplica,
		Status:  operatorv1.ConditionUnknown,
		Message: "",
		Reason:  "CheckDone",
	}

	ingress, err := r.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		cond.Message = err.Error()
		cond.Reason = "CheckFailed"
		return conditions.SetCondition(ctx, r.arocli, cond, r.role)
	}

	if ingress.Spec.Replicas != nil && *ingress.Spec.Replicas < 1 {
		var minimumReplica int32 = 1
		ingress.Spec.Replicas = &minimumReplica
		_, err := r.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ctx, ingress, metav1.UpdateOptions{})
		if err != nil {
			cond.Message = err.Error()
			cond.Reason = "RescaleFailed"
			return conditions.SetCondition(ctx, r.arocli, cond, r.role)
		}
		cond.Message = "RescaleSucceeded"
	}

	return conditions.SetCondition(ctx, r.arocli, cond, r.role)
}
