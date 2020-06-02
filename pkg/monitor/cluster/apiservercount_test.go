package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitOpenshiftApiServerCount(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apiserver",
				Namespace: "openshift-apiserver",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberAvailable:        2,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apiserver-1",
				Namespace: "openshift-kube-apiserver",
				Labels: map[string]string{
					"app": "openshift-kube-apiserver",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apiserver-2",
				Namespace: "openshift-kube-apiserver",
				Labels: map[string]string{
					"app": "openshift-kube-apiserver",
				},
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("apiserver.openshift.count", int64(2), map[string]string{})
	m.EXPECT().EmitGauge("apiserver.kube.count", int64(2), map[string]string{})

	ctx := context.Background()
	err := mon.emitAPIServerCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
