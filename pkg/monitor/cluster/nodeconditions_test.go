package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitNodeConditions(t *testing.T) {
	ctx := context.Background()

	provSpec, err := json.Marshal(azureproviderv1beta1.AzureMachineProviderSpec{})
	if err != nil {
		t.Fatal(err)
	}

	spotProvSpec, err := json.Marshal(azureproviderv1beta1.AzureMachineProviderSpec{
		SpotVMOptions: &azureproviderv1beta1.SpotVMOptions{},
	})
	if err != nil {
		t.Fatal(err)
	}

	kubeletVersion := "v1.17.1+9d33dd3"

	for _, tt := range []struct {
		name   string
		cli    kubernetes.Interface
		maocli maoclient.Interface
		mocks  func(*mock_metrics.MockInterface)
	}{
		{
			name: "basic",
			cli: fake.NewSimpleClientset(
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "aro-master-0",
						Annotations: map[string]string{"machine.openshift.io/machine": "openshift-machine-api/master-0"},
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
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-1",
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
				},
			),
			maocli: maofake.NewSimpleClientset(
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
			),
			mocks: func(m *mock_metrics.MockInterface) {
				m.EXPECT().EmitGauge("node.count", int64(2), map[string]string{})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName": "aro-master-0",
					"status":   "True",
					"type":     "MemoryPressure",
				})
				m.EXPECT().EmitGauge("node.conditions", int64(1), map[string]string{
					"nodeName": "aro-master-1",
					"status":   "False",
					"type":     "Ready",
				})

				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-master-0",
					"kubeletVersion": kubeletVersion,
				})
				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-master-1",
					"kubeletVersion": kubeletVersion,
				})
			},
		},
		{
			name: "spot VM",
			cli: fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "aro-spot-0",
					Annotations: map[string]string{"machine.openshift.io/machine": "openshift-machine-api/spot-0"},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionUnknown,
						},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: kubeletVersion,
					},
				},
			}),
			maocli: maofake.NewSimpleClientset(&machinev1beta1.Machine{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: spotProvSpec,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift-machine-api/spot-0",
					Namespace: "openshift-machine-api",
				},
			}),
			mocks: func(m *mock_metrics.MockInterface) {
				m.EXPECT().EmitGauge("node.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("node.kubelet.version", int64(1), map[string]string{
					"nodeName":       "aro-spot-0",
					"kubeletVersion": kubeletVersion,
				})
			},
		},
	} {
		controller := gomock.NewController(t)
		defer controller.Finish()

		m := mock_metrics.NewMockInterface(controller)

		mon := &Monitor{
			cli:    tt.cli,
			maocli: tt.maocli,
			log:    logrus.NewEntry(logrus.StandardLogger()),
			m:      m,
		}

		tt.mocks(m)

		err := mon.emitNodeConditions(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}
