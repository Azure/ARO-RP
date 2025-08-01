package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/pkg/kubelet/events"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitWorkloadStatuses(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	type testCase struct {
		name    string
		objects []runtime.Object
		mocks   func(*mock_metrics.MockEmitter)
		wantErr bool
	}

	for _, tt := range []*testCase{
		{
			name: "all workloads healthy",
			objects: []runtime.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "openshift"}},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "d1", Namespace: "openshift"},
					Status:     appsv1.DeploymentStatus{Replicas: 1, AvailableReplicas: 1},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("pod.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("daemonset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("deployment.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("replicaset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("statefulset.count", int64(0), map[string]string{})
			},
		},
		{
			name: "unhealthy workloads in openshift namespace",
			objects: []runtime.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "openshift"}},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "openshift"},
					Spec:       corev1.PodSpec{NodeName: "test-node"},
					Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}},
				},
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{Name: "ds1", Namespace: "openshift"},
					Status:     appsv1.DaemonSetStatus{DesiredNumberScheduled: 2, NumberAvailable: 1},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "d1", Namespace: "openshift"},
					Status:     appsv1.DeploymentStatus{Replicas: 2, AvailableReplicas: 1},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{Name: "rs1", Namespace: "openshift"},
					Status:     appsv1.ReplicaSetStatus{Replicas: 2, AvailableReplicas: 1},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "ss1", Namespace: "openshift"},
					Status:     appsv1.StatefulSetStatus{Replicas: 2, ReadyReplicas: 1},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("pod.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("daemonset.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("deployment.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("replicaset.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("statefulset.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("pod.conditions", int64(1), gomock.Any())
				m.EXPECT().EmitGauge("daemonset.statuses", int64(1), gomock.Any())
				m.EXPECT().EmitGauge("deployment.statuses", int64(1), gomock.Any())
				m.EXPECT().EmitGauge("replicaset.statuses", int64(1), gomock.Any())
				m.EXPECT().EmitGauge("statefulset.statuses", int64(1), gomock.Any())
			},
		},
		{
			name: "unhealthy workloads in customer namespace are ignored",
			objects: []runtime.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "customer-test"}},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "d1", Namespace: "customer-test"},
					Status:     appsv1.DeploymentStatus{Replicas: 2, AvailableReplicas: 1},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("pod.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("daemonset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("deployment.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("replicaset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("statefulset.count", int64(0), map[string]string{})
			},
		},
		{
			name: "pod with high restart count in openshift namespace",
			objects: []runtime.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "openshift-monitoring"}},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "p-restarts", Namespace: "openshift-monitoring"},
					Status:     corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{RestartCount: 20}}},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("pod.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("daemonset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("deployment.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("replicaset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("statefulset.count", int64(0), map[string]string{})
				m.EXPECT().EmitGauge("pod.restartcounter", int64(20), map[string]string{
					"name":      "p-restarts",
					"namespace": "openshift-monitoring",
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientGoFake := fake.NewSimpleClientset()
			// The fake client builder needs to know about the object types to return them
			scheme := runtime.NewScheme()
			appsv1.AddToScheme(scheme)
			corev1.AddToScheme(scheme)
			controllerRuntimeFake := fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			// Populate the client-go fake client with the objects for the second phase of the check
			for _, o := range tt.objects {
				clientGoFake.Tracker().Add(o)
			}

			mon := &Monitor{
				cli:          clientGoFake,
				ocpclientset: controllerRuntimeFake,
				m:            m,
				log:          log,
			}

			if tt.mocks != nil {
				tt.mocks(m)
			}

			if err := mon.emitWorkloadStatuses(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Monitor.emitWorkloadStatuses() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
		&corev1.Pod{ // no metrics expected - succeeded pod
			ObjectMeta: metav1.ObjectMeta{
				Name:      "succeeded-pod",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		&corev1.Pod{ // no metrics expected - preempted pod
			ObjectMeta: metav1.ObjectMeta{
				Name:      "preempted-pod",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				Reason: events.PreemptContainer,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

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
		&corev1.Pod{ // oomkilled pod
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oomkilled-pod1",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "oom-killed-cntr",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "CrashLoopBackOff",
							},
						},
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:   "OOMKilled",
								ExitCode: 137,
							},
						},
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node-name",
			},
		},
		&corev1.Pod{ // no metrics expected - succeeded pod
			ObjectMeta: metav1.ObjectMeta{
				Name:      "succeeded-pod",
				Namespace: "openshift",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
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
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("pod.containerstatuses", int64(1), map[string]string{
		"name":                 "name",
		"namespace":            "openshift",
		"nodeName":             "fake-node-name",
		"containername":        "containername",
		"reason":               "ImagePullBackOff",
		"lastTerminationState": "",
	})
	m.EXPECT().EmitGauge("pod.containerstatuses", int64(1), map[string]string{
		"name":                 "oomkilled-pod1",
		"namespace":            "openshift",
		"nodeName":             "fake-node-name",
		"containername":        "oom-killed-cntr",
		"reason":               "CrashLoopBackOff",
		"lastTerminationState": "OOMKilled",
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
				Namespace: "openshift",
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

	m := mock_metrics.NewMockEmitter(controller)

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
		"namespace": "openshift",
	})

	ps, _ := cli.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	mon._emitPodContainerRestartCounter(ps)

	// Matches the number of emitted messages
	assert.Len(t, hook.Entries, 3)

	// the order of the log entries does not seem to be stable, so testing one entry only
	// and no test for specific values, except for the metric

	x := hook.LastEntry()
	assert.NotEmpty(t, x.Data["name"])
	assert.NotEmpty(t, x.Data["namespace"])
	assert.Equal(t, "pod.restartcounter", x.Data["metric"])
}
