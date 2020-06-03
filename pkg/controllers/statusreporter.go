package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type StatusReporter struct {
	arocli aroclient.AroV1alpha1Interface
	name   string
	log    *logrus.Entry
}

var (
	emptyConditions = []status.Condition{
		{
			Type:    aro.InternetReachable,
			Status:  corev1.ConditionUnknown,
			Reason:  "",
			Message: "",
		},
	}
)

func NewStatusReporter(log *logrus.Entry, arocli aroclient.AroV1alpha1Interface, name string) *StatusReporter {
	return &StatusReporter{
		log:    log.WithField("manager", "StatusReporter"),
		arocli: arocli,
		name:   name,
	}
}

func (r *StatusReporter) SetNoInternetConnection(ctx context.Context, sitesNotAvailable map[string]string) error {
	time := metav1.Now()
	msg := ""
	for k, v := range sitesNotAvailable {
		msg += "[" + k + "] " + v + "\n"
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		co, err := r.arocli.Clusters().Get(r.name, v1.GetOptions{})
		if err != nil {
			return err
		}

		co.Status.Conditions.SetCondition(status.Condition{
			Type:               aro.InternetReachable,
			Status:             corev1.ConditionFalse,
			Message:            msg,
			Reason:             "CheckFailed",
			LastTransitionTime: time})

		setStaticStatus(&co.Status)

		_, err = r.arocli.Clusters().UpdateStatus(co)
		return err
	})
}

func (r *StatusReporter) SetInternetConnected(ctx context.Context) error {
	time := metav1.Now()
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		co, err := r.arocli.Clusters().Get(r.name, v1.GetOptions{})
		if err != nil {
			return err
		}

		co.Status.Conditions.SetCondition(status.Condition{
			Type:               aro.InternetReachable,
			Status:             corev1.ConditionTrue,
			Message:            "Outgoing connection successful.",
			Reason:             "CheckDone",
			LastTransitionTime: time})

		setStaticStatus(&co.Status)

		_, err = r.arocli.Clusters().UpdateStatus(co)
		return err
	})
}

func setStaticStatus(status *aro.ClusterStatus) {
	if len(status.RelatedObjects) == 0 {
		status.RelatedObjects = []corev1.ObjectReference{
			{Kind: "Secret", Name: "pull-secret", Namespace: "openshift-config"},
			{Kind: "Namespace", Name: "openshift-azure-logging"},
			{Kind: "ConfigMap", Name: "fluent-config", Namespace: "openshift-azure-logging"},
			{Kind: "ServiceAccount", Name: "geneva", Namespace: "openshift-azure-logging"},
			{Kind: "SecurityContextConstraints", Name: "privileged-genevalogging"},
			{Kind: "DaemonSet", Name: "mdsd", Namespace: "openshift-azure-logging"},
		}
	}
	status.OperatorVersion = version.GitCommit
}
