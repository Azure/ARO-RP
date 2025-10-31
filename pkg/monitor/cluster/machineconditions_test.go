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

// Phase constants
const (
	phaseRunning      = "Running"
	phaseFailed       = "Failed"
	phaseProvisioning = "Provisioning"
	phaseDeleting     = "Deleting"
	phaseDeleted      = "Deleted"
)

func TestEmitMachineConditions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		machines    []client.Object
		wantEmitted func(m *mock_metrics.MockEmitter)
	}{
		{
			name: "master machines - unexpected condition should emit conditions",
			machines: []client.Object{
				newTestMachine(t, "aro-master-0", "master", "", phaseFailed, "Ready", corev1.ConditionFalse, "Machine not ready", false),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 1)
				expectMachineCondition(m, "aro-master-0", "False", "Ready", "master", "", "false")
				expectPhaseCount(m, phaseFailed, 1)
			},
		},
		{
			name: "worker spot machine - unexpected condition should emit conditions with spot information",
			machines: []client.Object{
				newTestMachine(t, "aro-worker-spot-failed", "worker", "spot-workers-failed", phaseFailed, "Ready", corev1.ConditionFalse, "Machine failed", true),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 1)
				expectMachineCondition(m, "aro-worker-spot-failed", "False", "Ready", "worker", "spot-workers-failed", "true")
				expectPhaseCount(m, phaseFailed, 1)
			},
		},
		{
			name: "different phases - should emit accurate phase counts",
			machines: []client.Object{
				newTestMachine(t, "aro-master-running", "master", "", phaseRunning, "Ready", corev1.ConditionTrue, "Machine is ready", false),
				newTestMachine(t, "aro-worker-provisioning", "worker", "workers", phaseProvisioning, "Ready", corev1.ConditionFalse, "Provisioning in progress", false),
				newTestMachine(t, "aro-worker-deleting", "worker", "workers", phaseDeleting, "Ready", corev1.ConditionFalse, "Machine is being deleted", false),
				newTestMachine(t, "aro-worker-deleted", "worker", "workers-old", phaseDeleted, "Ready", corev1.ConditionFalse, "Machine has been deleted", false),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 4)
				expectMachineCondition(m, "aro-master-running", "True", "Ready", "master", "", "false")
				expectMachineCondition(m, "aro-worker-provisioning", "False", "Ready", "worker", "workers", "false")
				expectMachineCondition(m, "aro-worker-deleting", "False", "Ready", "worker", "workers", "false")
				expectMachineCondition(m, "aro-worker-deleted", "False", "Ready", "worker", "workers-old", "false")
				expectPhaseCount(m, phaseRunning, 1)
				expectPhaseCount(m, phaseProvisioning, 1)
				expectPhaseCount(m, phaseDeleting, 1)
				expectPhaseCount(m, phaseDeleted, 1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mon, m := setupTestMonitor(t, tt.machines)
			tt.wantEmitted(m)

			err := mon.emitMachineConditions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// setupTestMonitor creates a Monitor instance with a fake client and mock metrics emitter
func setupTestMonitor(t *testing.T, machines []client.Object) (*Monitor, *mock_metrics.MockEmitter) {
	t.Helper()

	controller := gomock.NewController(t)
	t.Cleanup(func() { controller.Finish() })

	m := mock_metrics.NewMockEmitter(controller)
	_, log := testlog.New()

	scheme := kruntime.NewScheme()
	_ = machinev1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	ocpclientset := clienthelper.NewWithClient(log, fake.
		NewClientBuilder().
		WithScheme(scheme).
		WithObjects(machines...).
		Build())

	mon := &Monitor{
		log:          log,
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   1,
	}

	return mon, m
}

// newTestMachine creates a Machine object for testing
func newTestMachine(t *testing.T, name, role, machineset, phase, conditionType string, conditionStatus corev1.ConditionStatus, conditionMessage string, isSpot bool) *machinev1beta1.Machine {
	t.Helper()

	labels := make(map[string]string)
	if role != "" {
		labels[machineRoleLabelKey] = role
	}
	if machineset != "" {
		labels[machinesetLabelKey] = machineset
	}

	var providerSpec machinev1beta1.ProviderSpec
	if isSpot {
		providerSpec = validMachineProviderSpecSpotVM(t)
	} else {
		providerSpec = validMachineProviderSpec(t)
	}

	machine := &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "openshift-machine-api",
			Labels:    labels,
		},
		Status: machinev1beta1.MachineStatus{
			Conditions: machinev1beta1.Conditions{
				{
					Type:    machinev1beta1.ConditionType(conditionType),
					Status:  conditionStatus,
					Message: conditionMessage,
				},
			},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: providerSpec,
		},
	}

	if phase != "" {
		machine.Status.Phase = &phase
	}

	return machine
}

// expectMachineCount sets up expectation for machine.count metric
func expectMachineCount(m *mock_metrics.MockEmitter, count int64) {
	m.EXPECT().EmitGauge("machine.count", count, map[string]string{})
}

// expectMachineCondition sets up expectation for machine.conditions metric
func expectMachineCondition(m *mock_metrics.MockEmitter, machineName, status, conditionType, role, machineset, spotInstance string) {
	m.EXPECT().EmitGauge("machine.conditions", int64(1), map[string]string{
		"machineName":  machineName,
		"status":       status,
		"type":         conditionType,
		"spotInstance": spotInstance,
		"role":         role,
		"machineset":   machineset,
	})
}

// expectPhaseCount sets up expectation for machine.count.phase metric
func expectPhaseCount(m *mock_metrics.MockEmitter, phase string, count int64) {
	m.EXPECT().EmitGauge("machine.count.phase", count, map[string]string{
		"phase": phase,
	})
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
