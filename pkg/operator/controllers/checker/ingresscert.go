// Implements a check that provides detail on potentially faulty or customised
// IngressController configurations on the default controller.
//
// Included checks are:
//  - existence of custom ingress certificate
//  - existence of default ingresscontroller

package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

type IngressCertificateChecker struct {
	arocli      aroclient.Interface
	operatorcli operatorclient.Interface
	configcli   configclient.Interface

	role string
}

func NewIngressCertificateChecker(log *logrus.Entry, arocli aroclient.Interface, operatorcli operatorclient.Interface, configcli configclient.Interface, role string) *IngressCertificateChecker {
	return &IngressCertificateChecker{
		arocli:      arocli,
		operatorcli: operatorcli,
		configcli:   configcli,
	}
}

func (r *IngressCertificateChecker) Name() string {
	return "IngressCertificateChecker"
}

func (r *IngressCertificateChecker) Check(ctx context.Context) error {
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.DefaultIngressCertificate,
		Status:  operatorv1.ConditionUnknown,
		Message: "",
		Reason:  "CheckDone",
	}

	cv, err := r.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		cond.Message = err.Error()
		cond.Reason = "CheckFailed"
		return conditions.SetCondition(ctx, r.arocli, cond, r.role)
	}

	ingress, err := r.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		cond.Message = err.Error()
		cond.Reason = "CheckFailed"
		return conditions.SetCondition(ctx, r.arocli, cond, r.role)
	}

	if ingress.Spec.DefaultCertificate == nil {
		cond.Status = operatorv1.ConditionFalse
		cond.Message = "Ingress has no certificate yet"
	} else if ingress.Spec.DefaultCertificate.Name != string(cv.Spec.ClusterID)+"-ingress" {
		cond.Status = operatorv1.ConditionFalse
		cond.Message = "Custom ingress certificate in use: " + ingress.Spec.DefaultCertificate.Name
	} else {
		cond.Status = operatorv1.ConditionTrue
		cond.Message = "Default ingress certificate in use"
	}

	return conditions.SetCondition(ctx, r.arocli, cond, r.role)
}
