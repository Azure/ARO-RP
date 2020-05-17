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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

type StatusReporter struct {
	arocli aroclient.AroV1alpha1Interface
	name   types.NamespacedName
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

func NewStatusReporter(log *logrus.Entry, arocli aroclient.AroV1alpha1Interface, namespace, name string) *StatusReporter {
	return &StatusReporter{
		log:    log.WithField("manager", "StatusReporter"),
		arocli: arocli,
		name:   types.NamespacedName{Name: name, Namespace: namespace},
	}
}

func (r *StatusReporter) SetNoInternetConnection(ctx context.Context, connectionErr error) error {
	time := metav1.Now()
	msg := "Outgoing connection failed"
	if connectionErr != nil {
		msg += ": " + connectionErr.Error()
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		co, err := r.arocli.Clusters(r.name.Namespace).Get(r.name.Name, v1.GetOptions{})
		if err != nil {
			return err
		}

		co.Status.Conditions.SetCondition(status.Condition{
			Type:               aro.InternetReachable,
			Status:             corev1.ConditionFalse,
			Message:            msg,
			Reason:             "CheckFailed",
			LastTransitionTime: time})
		if len(co.Status.RelatedObjects) == 0 {
			co.Status.RelatedObjects = newRelatedObjects(r.name.Namespace)
		}

		_, err = r.arocli.Clusters(r.name.Namespace).Update(co)
		return err
	})
}

func (r *StatusReporter) SetInternetConnected(ctx context.Context) error {
	time := metav1.Now()
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		co, err := r.arocli.Clusters(r.name.Namespace).Get(r.name.Name, v1.GetOptions{})
		if err != nil {
			return err
		}

		co.Status.Conditions.SetCondition(status.Condition{
			Type:               aro.InternetReachable,
			Status:             corev1.ConditionTrue,
			Message:            "Outgoing connection successful.",
			Reason:             "CheckDone",
			LastTransitionTime: time})
		if len(co.Status.RelatedObjects) == 0 {
			co.Status.RelatedObjects = newRelatedObjects(r.name.Namespace)
		}

		_, err = r.arocli.Clusters(r.name.Namespace).Update(co)
		return err
	})
}

func newRelatedObjects(namespace string) []corev1.ObjectReference {
	return []corev1.ObjectReference{
		{Kind: "Namespace", Name: namespace},
		{Kind: "Secret", Name: "pull-secret", Namespace: "openshift-config"},
	}
}
