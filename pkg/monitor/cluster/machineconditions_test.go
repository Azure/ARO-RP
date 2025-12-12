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
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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

func TestEmitMachinePhase(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		machines    []client.Object
		wantEmitted func(m *mock_metrics.MockEmitter)
	}{
		{
			name: "master machine - should emit phase",
			machines: []client.Object{
				newTestMachine(t, "aro-master-0", "master", "", phaseFailed, false),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 1)
				expectMachineCondition(m, "aro-master-0", "master", "", "Failed", "false")
			},
		},
		{
			name: "worker spot machine - should emit phase with spot information",
			machines: []client.Object{
				newTestMachine(t, "aro-worker-spot-failed", "worker", "spot-workers-failed", phaseFailed, true),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 1)
				expectMachineCondition(m, "aro-worker-spot-failed", "worker", "spot-workers-failed", "Failed", "true")
			},
		},
		{
			name: "different phases - should emit accurate phases",
			machines: []client.Object{
				newTestMachine(t, "aro-master-running", "master", "", phaseRunning, false),
				newTestMachine(t, "aro-worker-provisioning", "worker", "worker-machineset", phaseProvisioning, false),
				newTestMachine(t, "aro-worker-deleting", "worker", "worker-machineset", phaseDeleting, false),
				newTestMachine(t, "aro-worker-deleted", "worker", "worker-machineset", phaseDeleted, false),
			},
			wantEmitted: func(m *mock_metrics.MockEmitter) {
				expectMachineCount(m, 4)
				expectMachineCondition(m, "aro-master-running", "master", "", phaseRunning, "false")
				expectMachineCondition(m, "aro-worker-provisioning", "worker", "worker-machineset", phaseProvisioning, "false")
				expectMachineCondition(m, "aro-worker-deleting", "worker", "worker-machineset", phaseDeleting, "false")
				expectMachineCondition(m, "aro-worker-deleted", "worker", "worker-machineset", phaseDeleted, "false")
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
func newTestMachine(t *testing.T, name, role, machineset, phase string, isSpot bool) *machinev1beta1.Machine {
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
			Phase: pointerutils.ToPtr(phase),
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

// expectMachineCondition sets up expectation for machine.phase metric
func expectMachineCondition(m *mock_metrics.MockEmitter, machineName, role, machineset, phase string, spotInstance string) {
	m.EXPECT().EmitGauge("machine.phase", int64(1), map[string]string{
		"machineName":  machineName,
		"phase":        phase,
		"spotInstance": spotInstance,
		"role":         role,
		"machineset":   machineset,
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
