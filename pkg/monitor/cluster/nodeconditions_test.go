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

	provSpec, err := json.Marshal(machinev1beta1.AzureMachineProviderSpec{})
	if err != nil {
		t.Fatal(err)
	}

	kubeletVersion := "v1.17.1+9d33dd3"

	cli := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-0",
			Annotations: map[string]string{
				"machine.openshift.io/machine": "openshift-machine-api/master-0",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionTrue,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: kubeletVersion,
			},
		},
	}, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-1",
			Annotations: map[string]string{
				"machine.openshift.io/machine": "openshift-machine-api/master-1",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: kubeletVersion,
			},
		},
	})
	machineclient := machinefake.NewSimpleClientset(
		&machinev1beta1.Machine{
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &kruntime.RawExtension{
						Raw: provSpec,
					},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "openshift-machine-api/master-0",
				Namespace: "openshift-machine-api",
			},
		},
		&machinev1beta1.Machine{
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &kruntime.RawExtension{
						Raw: provSpec,
					},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "openshift-machine-api/master-1",
				Namespace: "openshift-machine-api",
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		cli:    cli,
		maocli: machineclient,
		m:      m,
	}

	m.EXPECT().EmitGauge("node.count", int64(2), map[string]string{})
	m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
		"nodeName":     "aro-master-0",
		"status":       "True",
		"type":         "MemoryPressure",
		"spotInstance": "false",
	})
	m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
		"nodeName":     "aro-master-1",
		"status":       "False",
		"type":         "Ready",
		"spotInstance": "false",
	})

	m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
		"nodeName":       "aro-master-0",
		"kubeletVersion": kubeletVersion,
	})
	m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
		"nodeName":       "aro-master-1",
		"kubeletVersion": kubeletVersion,
	})

	err = mon.emitNodeConditions(ctx)
	if err != nil {
		t.Fatal(err)
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
