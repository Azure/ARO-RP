package advisor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

type IngressCertificateChecker struct {
	log *logrus.Entry

	arocli      aroclient.Interface
	operatorcli operatorclient.Interface
	configcli   configclient.Interface

	role string
}

func NewIngressCertificateChecker(log *logrus.Entry, arocli aroclient.Interface, operatorcli operatorclient.Interface, configcli configclient.Interface, role string) *IngressCertificateChecker {
	return &IngressCertificateChecker{
		log:    log,
		arocli: arocli,
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
		updateFailedCondition(cond, err)
		return conditions.SetAdvisorCondition(ctx, r.arocli, cond, r.role)
	}

	ingress, err := r.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		cond.Message = "default ingress not found"
		updateFailedCondition(cond, err)
		return conditions.SetAdvisorCondition(ctx, r.arocli, cond, r.role)
	} else if err != nil {
		updateFailedCondition(cond, err)
		return conditions.SetAdvisorCondition(ctx, r.arocli, cond, r.role)
	}

	if ingress.Spec.DefaultCertificate.Name != string(cv.Spec.ClusterID)+"-ingress" {
		cond.Status = operatorv1.ConditionFalse
		cond.Message = "Custom ingress certificate in use: " + ingress.Spec.DefaultCertificate.Name
	} else {
		cond.Status = operatorv1.ConditionTrue
		cond.Message = "Default ingress certificate in use"
	}

	return conditions.SetAdvisorCondition(ctx, r.arocli, cond, r.role)
}

func updateFailedCondition(cond *operatorv1.OperatorCondition, err error) {
	cond.Status = operatorv1.ConditionUnknown
	if tErr, ok := err.(*api.CloudError); ok {
		cond.Message = tErr.Message
	} else {
		cond.Message = err.Error()
	}
	cond.Reason = "CheckFailed"
}
