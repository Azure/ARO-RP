package statusreporter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aro "github.com/Azure/ARO-RP/operator/api/v1alpha1"
)

var (
	log = logf.Log.WithName("statusreporter")
)

type StatusReporter struct {
	client client.Client
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
		{
			Type:    aro.ClusterSupportable,
			Status:  corev1.ConditionUnknown,
			Reason:  "",
			Message: "",
		},
	}
	supportable = map[bool]status.Condition{
		true: {
			Type:    aro.ClusterSupportable,
			Status:  corev1.ConditionTrue,
			Message: "Cluster is supportable.",
			Reason:  "AllChecksDone"},
		false: {
			Type:    aro.ClusterSupportable,
			Status:  corev1.ConditionFalse,
			Message: "Cluster is NOT supportable.",
			Reason:  "SomeChecksFailed"},
	}
)

func NewStatusReporter(client_ client.Client, namespace, name string) *StatusReporter {
	return &StatusReporter{
		client: client_,
		name:   types.NamespacedName{Name: name, Namespace: namespace},
	}
}

func (r *StatusReporter) setSupportable(status_ *aro.ClusterStatus, now metav1.Time) {
	isSupportable := true
	reason := supportable[true].Reason
	for _, cond := range status_.Conditions {
		if cond.Type != aro.ClusterSupportable && !cond.IsTrue() {
			isSupportable = false
			reason = status.ConditionReason(cond.Type + "Failed")
		}
	}

	sup := status_.Conditions.GetCondition(aro.ClusterSupportable)
	newSup := supportable[isSupportable]

	if isSupportable != sup.IsTrue() {
		newSup.LastTransitionTime = now
		log.Info("cluster supportable condition transition", "current", sup, "new", newSup)
	}
	newSup.Reason = reason
	status_.Conditions.SetCondition(newSup)
}

func (r *StatusReporter) SetNoInternetConnection(ctx context.Context, connectionErr error) error {
	co := &aro.Cluster{}
	err := r.client.Get(ctx, r.name, co)
	if apierrors.IsNotFound(err) {
		co = r.newCluster()
		err = r.client.Create(ctx, co)
	}
	if err != nil && !apierrors.IsNotFound(err) {
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

	r.setSupportable(&co.Status, time)
	// TODO handle conflicts
	return r.client.Status().Update(ctx, co)
}

func (r *StatusReporter) SetInternetConnected(ctx context.Context) error {
	co := &aro.Cluster{}
	err := r.client.Get(ctx, r.name, co)
	if apierrors.IsNotFound(err) {
		co = r.newCluster()
		err = r.client.Create(ctx, co)
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

	r.setSupportable(&co.Status, time)
	// TODO handle conflicts
	log.Info("updating cluster status")
	return r.client.Status().Update(ctx, co)
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
