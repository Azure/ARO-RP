package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
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

func TestEmitPodContainerRestartCounter(t *testing.T) {

	cli := fake.NewSimpleClientset(
		&corev1.Pod{ // #1 metrics and log entry expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podname1",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: 42,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // #2 no metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podname2",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: restartCounterThreshold - 1,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // #3 metrics and log entry expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podname3",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: restartCounterThreshold,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // #4 no metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podname4",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: 0,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // #5 no metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "not-system-namespace",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "containername",
						RestartCount: 42,
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // #6 Multi-container pod
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-container-pod",
				Namespace: "openshift-test",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "firstcontainer",
						RestartCount: restartCounterThreshold,
					},
					{
						Name:         "secondcontainer",
						RestartCount: restartCounterThreshold,
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
		cli:       cli,
		m:         m,
		hourlyRun: true,
	}
	logger, hook := test.NewNullLogger()
	log := logrus.NewEntry(logger)
	mon.log = log

	m.EXPECT().EmitGauge("pod.restartcounter", int64(42), map[string]string{
		"name":      "podname1",
		"namespace": "openshift",
	})

	// Expecting data for 'podname2' to be dropped

	m.EXPECT().EmitGauge("pod.restartcounter", int64(restartCounterThreshold), map[string]string{
		"name":      "podname3",
		"namespace": "openshift",
	})

	// Expecting data for 'podname4' to be dropped

	m.EXPECT().EmitGauge("pod.restartcounter", int64(restartCounterThreshold*2), map[string]string{
		"name":      "multi-container-pod",
		"namespace": "openshift-test",
	})

	ps, _ := cli.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	mon._emitPodContainerRestartCounter(ps)

	// Matches the number of emitted messages
	assert.Equal(t, 3, len(hook.Entries))

	// the order of the log entries does not seem to be stable, so testing one entry only
	// and no test for specific values, except for the metric

	x := hook.LastEntry()
	assert.NotEmpty(t, x.Data["name"])
	assert.NotEmpty(t, x.Data["namespace"])
	assert.Equal(t, "pod.restartcounter", x.Data["metric"])
}
