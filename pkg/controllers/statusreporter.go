package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

var (
	log = logf.Log.WithName("statusreporter")
)

type StatusReporter struct {
	arocli aroclient.AroV1alpha1Interface
	name   types.NamespacedName
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

func NewStatusReporter(arocli aroclient.AroV1alpha1Interface, namespace, name string) *StatusReporter {
	return &StatusReporter{
		arocli: arocli,
		name:   types.NamespacedName{Name: name, Namespace: namespace},
	}
}

func (r *StatusReporter) SetNoInternetConnection(ctx context.Context, connectionErr error) error {
	co, err := r.arocli.Clusters(r.name.Namespace).Get(r.name.Name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		co = r.newCluster()
		_, err = r.arocli.Clusters(r.name.Namespace).Create(co)
	}
	if err != nil {
		return err
	}

	time := metav1.Now()
	msg := "Outgoing connection failed"
	if connectionErr != nil {
		msg += ": " + connectionErr.Error()
	}
	co.Status.Conditions.SetCondition(status.Condition{
		Type:               aro.InternetReachable,
		Status:             corev1.ConditionFalse,
		Message:            msg,
		Reason:             "CheckFailed",
		LastTransitionTime: time})

	// TODO handle conflicts
	_, err = r.arocli.Clusters(r.name.Namespace).UpdateStatus(co)
	return err
}

func (r *StatusReporter) SetInternetConnected(ctx context.Context) error {
	co, err := r.arocli.Clusters(r.name.Namespace).Get(r.name.Name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		co = r.newCluster()
		_, err = r.arocli.Clusters(r.name.Namespace).Create(co)
	}
	if err != nil {
		return err
	}

	time := metav1.Now()
	co.Status.Conditions.SetCondition(status.Condition{
		Type:               aro.InternetReachable,
		Status:             corev1.ConditionTrue,
		Message:            "Outgoing connection successful.",
		Reason:             "CheckDone",
		LastTransitionTime: time})

	// TODO handle conflicts
	_, err = r.arocli.Clusters(r.name.Namespace).UpdateStatus(co)
	return err
}

func newRelatedObjects(namespace string) []corev1.ObjectReference {
	return []corev1.ObjectReference{
		{Kind: "Namespace", Name: namespace},
		{Kind: "Secret", Name: "pull-secret", Namespace: "openshift-config"},
	}
}

func (r *StatusReporter) newCluster() *aro.Cluster {
	co := &aro.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aro.openshift.io/v1alpha1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name.Name,
			Namespace: r.name.Namespace,
		},
		Spec: aro.ClusterSpec{},
		Status: aro.ClusterStatus{
			Conditions: emptyConditions,
		},
	}
	co.Status.RelatedObjects = newRelatedObjects(r.name.Namespace)
	return co
}
