package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitNodeConditions(t *testing.T) {
	ctx := context.Background()
	kubeletVersion := "v1.17.1+9d33dd3"

	for _, tt := range []struct {
		name        string
		nodes       []kruntime.Object
		machines    []kruntime.Object
		wantEmitted func(m *mock_metrics.MockEmitter)
	}{
		{
			name: "standard cluster",
			nodes: []kruntime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
						Annotations: map[string]string{
							"machine.openshift.io/machine": "openshift-machine-api/aro-master-0",
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
							"machine.openshift.io/machine": "openshift-machine-api/aro-master-1",
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
							"machine.openshift.io/machine": "openshift-machine-api/aro-master-2",
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
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-worker",
						Annotations: map[string]string{
							"machine.openshift.io/machine": "openshift-machine-api/aro-worker",
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
						Name: "aro-worker-spot",
						Annotations: map[string]string{
							"machine.openshift.io/machine": "openshift-machine-api/aro-worker-spot",
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
						Name: "aro-infra",
						Annotations: map[string]string{
							"machine.openshift.io/machine": "openshift-machine-api/aro-infra",
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
			},
			machines: []kruntime.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-0",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-1",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-2",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-worker",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-worker-spot",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpecSpotVM(t),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-infra",
						Namespace: "openshift-machine-api",
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("node.count", int64(6), map[string]string{})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-0",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-1",
					"status":       "True",
					"type":         "MemoryPressure",
					"spotInstance": "false",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-master-2",
					"status":       "True",
					"type":         "DiskPressure",
					"spotInstance": "false",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-worker",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-worker-spot",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "true",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName":     "aro-infra",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
				})

				for _, nodeName := range []string{
					"aro-master-0", "aro-master-1", "aro-master-2",
					"aro-worker", "aro-worker-spot", "aro-infra",
				} {
					m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
						"nodeName":       nodeName,
						"kubeletVersion": kubeletVersion,
					})
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.nodes...)
			maocli := machinefake.NewSimpleClientset(tt.machines...)

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			tt.wantEmitted(m)

			mon := &Monitor{
				cli:    cli,
				maocli: maocli,
				m:      m,
			}

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

func TestGetSpotInstances(t *testing.T) {
	ctx := context.Background()

	spotProvSpec, err := json.Marshal(machinev1beta1.AzureMachineProviderSpec{
		SpotVMOptions: &machinev1beta1.SpotVMOptions{},
	})
	if err != nil {
		t.Fatal(err)
	}

	provSpec, err := json.Marshal(machinev1beta1.AzureMachineProviderSpec{})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name                 string
		maocli               machineclient.Interface
		node                 corev1.Node
		expectedSpotInstance bool
	}{
		{
			name: "node is a spot instance",
			maocli: machinefake.NewSimpleClientset(&machinev1beta1.Machine{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: spotProvSpec,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-spot-0",
					Namespace: "openshift-machine-api",
				},
			}),
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-spot-0",
					Annotations: map[string]string{
						"machine.openshift.io/machine": "openshift-machine-api/spot-0",
					},
				},
			},
			expectedSpotInstance: true,
		},
		{
			name: "node is not a spot instance",
			maocli: machinefake.NewSimpleClientset(&machinev1beta1.Machine{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: provSpec,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-0",
					Namespace: "openshift-machine-api",
				},
			}),
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-master-0",
					Annotations: map[string]string{
						"machine.openshift.io/machine": "openshift-machine-api/master-0",
					},
				},
			},
			expectedSpotInstance: false,
		},
		{
			name: "node is missing annotation",
			maocli: machinefake.NewSimpleClientset(&machinev1beta1.Machine{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: provSpec,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-0",
					Namespace: "openshift-machine-api",
				},
			}),
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "aro-master-0",
					Annotations: map[string]string{},
				},
			},
			expectedSpotInstance: false,
		},
		{
			name: "malformed json in providerSpec",
			maocli: machinefake.NewSimpleClientset(&machinev1beta1.Machine{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(";df9j"),
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-spot-1",
					Namespace: "openshift-machine-api",
				},
			}),
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-spot-1",
					Annotations: map[string]string{
						"machine.openshift.io/machine": "openshift-machine-api/spot-0",
					},
				},
			},
			expectedSpotInstance: false,
		},
	} {
		controller := gomock.NewController(t)
		defer controller.Finish()

		mon := &Monitor{
			maocli: tt.maocli,
			log:    logrus.NewEntry(logrus.StandardLogger()),
		}

		_, isSpotInstance := mon.getSpotInstances(ctx)[tt.node.Name]
		if isSpotInstance != tt.expectedSpotInstance {
			t.Fatalf("test %s: isSpotInstance should be %t but got %t", tt.name, tt.expectedSpotInstance, isSpotInstance)
		}
	}
}
