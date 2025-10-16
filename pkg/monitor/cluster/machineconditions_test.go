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

func TestEmitMachineConditions(t *testing.T) {
	ctx := context.Background()
	phaseFailed := "Failed"

	for _, tt := range []struct {
		name        string
		machines    []client.Object
		wantEmitted func(m *mock_metrics.MockEmitter)
	}{
		{
			name: "master machines - unexpected condition should emit conditions",
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-master-0",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "master",
						},
					},
					Status: machinev1beta1.MachineStatus{
						Phase: &phaseFailed,
						Conditions: machinev1beta1.Conditions{
							{Type: "Ready", Status: corev1.ConditionFalse, Message: "Machine not ready"},
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validMachineProviderSpec(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("machine.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("machine.conditions", int64(1), map[string]string{
					"machineName":  "aro-master-0",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "false",
					"role":         "master",
					"machineset":   "",
				})
				m.EXPECT().EmitGauge("machine.count.phase", int64(1), map[string]string{
					"phase": "Failed",
				})
			},
		},
		{
			name: "worker spot machine - unexpected condition should emit conditions with spot information",
			machines: []client.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aro-worker-spot-failed",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							machineRoleLabelKey: "worker",
							machinesetLabelKey:  "spot-workers-failed",
						},
					},
					Status: machinev1beta1.MachineStatus{
						Phase: &phaseFailed,
						Conditions: machinev1beta1.Conditions{
							{Type: "Ready", Status: corev1.ConditionFalse, Message: "Machine failed"},
						},
					},
					Spec: machinev1beta1.MachineSpec{
						ProviderSpec: validMachineProviderSpecSpotVM(t),
					},
				},
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("machine.count", int64(1), map[string]string{})
				m.EXPECT().EmitGauge("machine.conditions", int64(1), map[string]string{
					"machineName":  "aro-worker-spot-failed",
					"status":       "False",
					"type":         "Ready",
					"spotInstance": "true",
					"role":         "worker",
					"machineset":   "spot-workers-failed",
				})
				m.EXPECT().EmitGauge("machine.count.phase", int64(1), map[string]string{
					"phase": "Failed",
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			_, log := testlog.New()

			scheme := kruntime.NewScheme()
			_ = machinev1beta1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.machines...).
				Build())

			mon := &Monitor{
				log:          log,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
			}

			tt.wantEmitted(m)

			err := mon.emitMachineConditions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func validMachineProviderSpec(t *testing.T) machinev1beta1.ProviderSpec {
	t.Helper()

	return buildAzureMachineProviderSpec(t, machinev1beta1.AzureMachineProviderSpec{})
}

func validMachineProviderSpecSpotVM(t *testing.T) machinev1beta1.ProviderSpec {
	t.Helper()

	return buildAzureMachineProviderSpec(t, machinev1beta1.AzureMachineProviderSpec{
		SpotVMOptions: &machinev1beta1.SpotVMOptions{},
	})
}

func buildAzureMachineProviderSpec(t *testing.T, amps machinev1beta1.AzureMachineProviderSpec) machinev1beta1.ProviderSpec {
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
