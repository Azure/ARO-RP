package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitNodeConditions(t *testing.T) {
	kubeletVersion := "v1.17.1+9d33dd3"

	for _, tt := range []struct {
		name        string
		nodes       []client.Object
		machines    []client.Object
		wantEmitted func(m *mock_metrics.MockEmitter)
	}{
		{
			name: "control plane - emits conditions only when unexpected",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-master-0",
						},
						Labels: map[string]string{
							masterRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
							{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
							{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-1",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-master-1",
						},
						Labels: map[string]string{
							masterRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
							{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
							{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-2",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-master-2",
						},
						Labels: map[string]string{
							masterRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
							{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
							{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-0",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "master",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-1",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "master",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-2",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "master",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(3), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-0",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "master",
					"machineset":   "",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-1",
					"status":       "True",
					"type":         "MemoryPressure",
					"spotInstance": "false",
					"role":         "master",
					"machineset":   "",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-2",
					"status":       "True",
					"type":         "DiskPressure",
					"spotInstance": "false",
					"role":         "master",
					"machineset":   "",
				})

				for _, nodeName := range []string{"aro-master-0", "aro-master-1", "aro-master-2"} {
					m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
						"nodeName":       nodeName,
						"kubeletVersion": kubeletVersion,
						"role":           "master",
					})
				}
			},
		},
		{
			name: "worker/infra nodes - emits spotVM and machineset information",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-worker",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-worker",
						},
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-worker-spot",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-worker-spot",
						},
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-infra",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-infra",
						},
						Labels: map[string]string{
							infraRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-worker",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
							machinesetLabelKey:  "workers",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-worker-spot",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
							machinesetLabelKey:  "spot-workers",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpecSpotVM(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-infra",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "infra",
							machinesetLabelKey:  "infras",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(2), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "infra"})

				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-worker",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "worker",
					"machineset":   "workers",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-worker-spot",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "true",
					"role":         "worker",
					"machineset":   "spot-workers",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-infra",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "infra",
					"machineset":   "infras",
				})

				for _, nodeName := range []string{"aro-worker", "aro-worker-spot"} {
					m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
						"nodeName":       nodeName,
						"kubeletVersion": kubeletVersion,
						"role":           "worker",
					})
				}

				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-infra",
					"kubeletVersion": kubeletVersion,
					"role":           "infra",
				})
			},
		},
		{
			name: "node missing machine - emits empty dimensions for machine values",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-impossible-node",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-impossible-node",
						},
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-impossible-node",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "",
					"machineset":   "",
				})

				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-impossible-node",
					"kubeletVersion": kubeletVersion,
					"role":           "",
				})
			},
		},
		{
			name:     "empty cluster - no nodes emits zero counts",
			nodes:    []client.Object{},
			machines: []client.Object{},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})
			},
		},
		{
			name: "healthy node - all conditions expected, no node.conditions emitted",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-healthy-worker",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-healthy-worker",
						},
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
							{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
							{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
							{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-healthy-worker",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
							machinesetLabelKey:  "workers",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})

				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-healthy-worker",
					"kubeletVersion": kubeletVersion,
					"role":           "worker",
				})
			},
		},
		{
			name: "node with no role labels - not counted as master nor worker nor infra",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-unlabeled-node",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-unlabeled-node",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-unlabeled-node",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})

				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "unknown"})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-unlabeled-node",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "worker",
					"machineset":   "",
				})
				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-unlabeled-node",
					"kubeletVersion": kubeletVersion,
					"role":           "worker",
				})
			},
		},
		{
			name: "node with no annotations - no machine match, empty role/machineset",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-no-annotation-node",
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})

				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-no-annotation-node",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "",
					"machineset":   "",
				})
				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-no-annotation-node",
					"kubeletVersion": kubeletVersion,
					"role":           "",
				})
			},
		},
		{
			name: "node with both worker AND infra labels - counted only once as worker",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-dual-label-node",
						Annotations: map[string]string{
							machineAnnotationKey: "openshift-machine-api/aro-dual-label-node",
						},
						Labels: map[string]string{
							workerRoleLabel: "",
							infraRoleLabel:  "",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
						},
						NodeInfo: corev1.NodeSystemInfo{
							KubeletVersion: kubeletVersion,
						},
					},
				},
			},
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-dual-label-node",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
							machinesetLabelKey:  "workers",
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "master"})
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{"role": "worker"})
				m.EXPECT().EmitGauge("node.count", int64(0), map[string]string{"role": "infra"})

				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-dual-label-node",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "worker",
					"machineset":   "workers",
				})
				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-dual-label-node",
					"kubeletVersion": kubeletVersion,
					"role":           "worker",
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			_, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithObjects(tt.nodes...).
				WithObjects(tt.machines...).
				Build())

			mon := &Monitor{
				log:          log,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
			}

			tt.wantEmitted(m)

			err := mon.emitNodeConditions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func validProviderSpec(t *testing.T) machinev1beta1.ProviderSpec {
	t.Helper()

	return buildAzureProviderSpec(t, machinev1beta1.AzureMachineProviderSpec{})
}

func validProviderSpecSpotVM(t *testing.T) machinev1beta1.ProviderSpec {
	t.Helper()

	return buildAzureProviderSpec(t, machinev1beta1.AzureMachineProviderSpec{
		SpotVMOptions: &machinev1beta1.SpotVMOptions{},
	})
}

func buildAzureProviderSpec(t *testing.T, amps machinev1beta1.AzureMachineProviderSpec) machinev1beta1.ProviderSpec {
	t.Helper()

	raw, err := json.Marshal(amps)
	if err != nil {
		t.Fatal(err)
	}

	return machinev1beta1.ProviderSpec{
		Value: &kruntime.RawExtension{
			Raw: raw,
		},
	}
}
