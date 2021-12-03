package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitPodConditions(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&corev1.Pod{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "openshift",
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   corev1.PodInitialized,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   corev1.PodScheduled,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   corev1.ContainersReady,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
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

	m.EXPECT().EmitGauge("pod.conditions", int64(1), map[string]string{
		"name":      "name",
		"namespace": "openshift",
		"nodeName":  "fake-node-name",
		"status":    "False",
		"type":      "ContainersReady",
	})
	m.EXPECT().EmitGauge("pod.conditions", int64(1), map[string]string{
		"name":      "name",
		"namespace": "openshift",
		"nodeName":  "fake-node-name",
		"status":    "False",
		"type":      "Initialized",
	})
	m.EXPECT().EmitGauge("pod.conditions", int64(1), map[string]string{
		"name":      "name",
		"namespace": "openshift",
		"nodeName":  "fake-node-name",
		"status":    "False",
		"type":      "PodScheduled",
	})
	m.EXPECT().EmitGauge("pod.conditions", int64(1), map[string]string{
		"name":      "name",
		"namespace": "openshift",
		"nodeName":  "fake-node-name",
		"status":    "False",
		"type":      "Ready",
	})

	ps, _ := cli.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	mon._emitPodConditions(ps)
}

func TestEmitPodContainerStatuses(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&corev1.Pod{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "containername",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ImagePullBackOff",
							},
						},
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
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

	m.EXPECT().EmitGauge("pod.containerstatuses", int64(1), map[string]string{
		"name":          "name",
		"namespace":     "openshift",
		"nodeName":      "fake-node-name",
		"containername": "containername",
		"reason":        "ImagePullBackOff",
	})

	ps, _ := cli.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	mon._emitPodContainerStatuses(ps)
}

func TestEmitPodRestartCounts(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&corev1.Pod{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: 43,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
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

	m.EXPECT().EmitGauge("pod.restartcounter", int64(43), map[string]string{
		"name":      "name",
		"namespace": "openshift",
	})

	ps, _ := cli.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	mon._emitPodContainerRestartCounter(ps)
}
