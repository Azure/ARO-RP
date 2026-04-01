package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestValidateZoneDistribution(t *testing.T) {
	for _, tt := range []struct {
		name    string
		items   map[string]string
		getZone func(string) string
		wantErr string
	}{
		{
			name: "valid - 3 items in 3 different zones",
			items: map[string]string{
				"item1": "1",
				"item2": "2",
				"item3": "3",
			},
			getZone: func(s string) string { return s },
		},
		{
			name: "invalid - only 2 items",
			items: map[string]string{
				"item1": "1",
				"item2": "2",
			},
			getZone: func(s string) string { return s },
			wantErr: "expected 3 items, got 2",
		},
		{
			name: "invalid - 4 items",
			items: map[string]string{
				"item1": "1",
				"item2": "2",
				"item3": "3",
				"item4": "1",
			},
			getZone: func(s string) string { return s },
			wantErr: "expected 3 items, got 4",
		},
		{
			name: "invalid - 3 items but only 2 unique zones",
			items: map[string]string{
				"item1": "1",
				"item2": "2",
				"item3": "2",
			},
			getZone: func(s string) string { return s },
			wantErr: "items must be spread across 3 different zones, found 2 zone(s)",
		},
		{
			name: "invalid - 3 items but all in same zone",
			items: map[string]string{
				"item1": "1",
				"item2": "1",
				"item3": "1",
			},
			getZone: func(s string) string { return s },
			wantErr: "items must be spread across 3 different zones, found 1 zone(s)",
		},
		{
			name:    "invalid - empty map",
			items:   map[string]string{},
			getZone: func(s string) string { return s },
			wantErr: "expected 3 items, got 0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateZoneDistribution(tt.items, tt.getZone)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestValidateClusterMachinesAndVMs(t *testing.T) {
	_, log := testlog.New()

	for _, tt := range []struct {
		name           string
		ocMachines     map[string]machineValidationData
		azureVMs       map[string]azureVMValidationData
		wantErrStrings []string
	}{
		{
			name: "valid - all machines match Azure VMs",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "1", vmSize: "Standard_D8s_v3"},
				"master-1": {zone: "2", vmSize: "Standard_D8s_v3"},
				"master-2": {zone: "3", vmSize: "Standard_D8s_v3"},
			},
		},
		{
			name: "invalid - machine not found in Azure",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "1", vmSize: "Standard_D8s_v3"},
				"master-1": {zone: "2", vmSize: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{"machine master-2 not found in Azure resources"},
		},
		{
			name: "invalid - zone mismatch",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "1", vmSize: "Standard_D8s_v3"},
				"master-1": {zone: "3", vmSize: "Standard_D8s_v3"}, // Wrong zone
				"master-2": {zone: "3", vmSize: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{"machine master-1 has zone 2 in its spec, however Azure VM is running in zone 3"},
		},
		{
			name: "invalid - size mismatch",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "1", vmSize: "Standard_D8s_v3"},
				"master-1": {zone: "2", vmSize: "Standard_D16s_v3"}, // Wrong size
				"master-2": {zone: "3", vmSize: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{"machine master-1 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM"},
		},
		{
			name: "invalid - multiple errors collected",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "2", vmSize: "Standard_D16s_v3"}, // Both wrong
				"master-1": {zone: "2", vmSize: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{
				"machine master-0 has zone 1 in its spec, however Azure VM is running in zone 2",
				"machine master-0 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM",
				"machine master-2 not found in Azure resources",
			},
		},
		{
			name: "invalid - zone and size mismatch for same machine",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMValidationData{
				"master-0": {zone: "1", vmSize: "Standard_D8s_v3"},
				"master-1": {zone: "3", vmSize: "Standard_D16s_v3"}, // Both wrong
				"master-2": {zone: "3", vmSize: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{
				"machine master-1 has zone 2 in its spec, however Azure VM is running in zone 3",
				"machine master-1 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM",
			},
		},
		{
			name:       "valid - empty maps",
			ocMachines: map[string]machineValidationData{},
			azureVMs:   map[string]azureVMValidationData{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterMachinesAndVMs(log, tt.ocMachines, tt.azureVMs)

			if len(tt.wantErrStrings) > 0 {
				if err == nil {
					t.Fatalf("expected error with messages %v, got nil", tt.wantErrStrings)
				}
				errStr := err.Error()
				for _, wantErrString := range tt.wantErrStrings {
					if !strings.Contains(errStr, wantErrString) {
						t.Errorf("expected error to contain %q, but got: %s", wantErrString, errStr)
					}
				}
			} else {
				utilerror.AssertErrorMessage(t, err, "")
			}
		})
	}
}

// TestGetClusterMachines tests the getClusterMachines function with various scenarios
func TestGetClusterMachines(t *testing.T) {
	ctx := context.Background()

	// Helper to create a machine object
	// phase can be empty string to create a machine with nil Phase
	// labelInstanceType can be empty string to create a machine without instance-type label
	createMachine := func(name, role, labelZone, specZone, vmSize, labelInstanceType, phase string) machinev1beta1.Machine {
		providerSpec := &machinev1beta1.AzureMachineProviderSpec{
			Zone:   &specZone,
			VMSize: vmSize,
		}
		providerSpecRaw, _ := json.Marshal(providerSpec)

		labels := map[string]string{
			machineLabelZone: labelZone,
		}
		if role != "" {
			labels[machineLabelClusterAPIRole] = role
		}
		if labelInstanceType != "" {
			labels[machineLabelInstanceType] = labelInstanceType
		}

		machine := machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: providerSpecRaw,
					},
				},
			},
		}

		if phase != "" {
			machine.Status.Phase = &phase
		}

		return machine
	}

	// Helper to encode machine list to bytes
	encodeMachineList := func(machines ...machinev1beta1.Machine) []byte {
		machineList := &machinev1beta1.MachineList{
			Items: machines,
		}
		b, _ := json.Marshal(machineList)
		return b
	}

	for _, tt := range []struct {
		name      string
		mocks     func(*mock_adminactions.MockKubeActions)
		wantErr   string
		wantCount int
	}{
		{
			name: "success - 3 master machines",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - filters non-master machines",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("worker-0", "worker", "1", "1", "Standard_D4s_v3", "Standard_D4s_v3", "Running"),
					createMachine("worker-1", "worker", "2", "2", "Standard_D4s_v3", "Standard_D4s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters
		},
		{
			name: "success - filters machines without role label",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-3", "", "1", "1", "Standard_D8s_v3", "", "Running"), // No role label
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters with proper role label
		},
		{
			name: "success - filters machines named master- but with wrong role",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-infra-0", "infra", "1", "1", "Standard_D8s_v3", "", "Running"), // Has "master" in name but wrong role
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters with role=master
		},
		{
			name: "success - includes machines with nil phase",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", ""), // nil phase
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - includes machines with non-Running phases",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Provisioning"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Failed"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - only 2 master machines",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			kubeActions := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(kubeActions)

			machines, err := getClusterMachines(ctx, kubeActions)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error to contain %q, got: %s", tt.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(machines) != tt.wantCount {
					t.Errorf("expected %d machines, got %d", tt.wantCount, len(machines))
				}
			}
		})
	}
}

func TestValidateClusterMachines(t *testing.T) {
	_, log := testlog.New()

	for _, tt := range []struct {
		name      string
		machines  map[string]machineValidationData
		wantErr   string
		wantCount int
	}{
		{
			name: "success - 3 machines all valid",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantCount: 3,
		},
		{
			name: "failure - not 3 machines",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "expected 3 machines, got 2",
		},
		{
			name: "failure - zone mismatch between label and spec",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-0 has a mismatch between label zone 1 and spec zone 2",
		},
		{
			name: "failure - multiple zone mismatches",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "master-0 has a mismatch",
		},
		{
			name: "failure - 3 masters but only 2 zones",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "items must be spread across 3 different zones, found 2 zone(s)",
		},
		{
			name: "failure - machine with nil phase",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-1 status phase is not Running, current phase is nil",
		},
		{
			name: "failure - machine with Provisioning phase",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Provisioning", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-1 status phase is not Running, current phase is Provisioning",
		},
		{
			name: "failure - machine with Failed phase",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Failed", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-2 status phase is not Running, current phase is Failed",
		},
		{
			name: "failure - multiple machines with wrong phase",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Deleting", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "", labelInstanceType: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "master-0 status phase is not Running",
		},
		{
			name: "failure - machine missing instance-type label",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: ""},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-1 has a mismatch between label instance-type <missing> and instance type defined in the spec Standard_D8s_v3",
		},
		{
			name: "failure - machine with mismatched instance-type label",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D16s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "machine master-1 has a mismatch between label instance-type Standard_D16s_v3 and instance type defined in the spec Standard_D8s_v3",
		},
		{
			name: "failure - combination of instance-type and zone mismatches",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: ""},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "master-0 has a mismatch",
		},
		{
			name: "failure - machines have different sizes",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D16s_v3", phase: "Running", labelInstanceType: "Standard_D16s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
			},
			wantErr: "has size",
		},
		{
			name: "failure - multiple machines with different sizes",
			machines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3", phase: "Running", labelInstanceType: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D16s_v3", phase: "Running", labelInstanceType: "Standard_D16s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D32s_v3", phase: "Running", labelInstanceType: "Standard_D32s_v3"},
			},
			wantErr: "has size",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			machines, err := validateClusterMachines(log, tt.machines)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error to contain %q, got: %s", tt.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(machines) != tt.wantCount {
					t.Errorf("expected %d machines, got %d", tt.wantCount, len(machines))
				}
			}
		})
	}
}

// TestValidateVMPowerState tests the validateVMPowerState function
func TestValidateVMPowerState(t *testing.T) {
	_, log := testlog.New()

	for _, tt := range []struct {
		name       string
		vmStatuses []string
		vmName     string
		wantErr    string
	}{
		{
			name: "success - correct statuses",
			vmStatuses: []string{
				"ProvisioningState/succeeded",
				"PowerState/running",
			},
			vmName: "master-0",
		},
		{
			name: "success - statuses in different order",
			vmStatuses: []string{
				"PowerState/running",
				"ProvisioningState/succeeded",
			},
			vmName: "master-0",
		},
		{
			name: "failure - only 1 status",
			vmStatuses: []string{
				"PowerState/running",
			},
			vmName:  "master-0",
			wantErr: "expected 2 statuses for VM master-0, but found 1: PowerState/running",
		},
		{
			name: "failure - 3 statuses",
			vmStatuses: []string{
				"ProvisioningState/succeeded",
				"PowerState/running",
				"ExtraStatus/unexpected",
			},
			vmName:  "master-0",
			wantErr: "expected 2 statuses for VM master-0, but found 3: ProvisioningState/succeeded, PowerState/running, ExtraStatus/unexpected",
		},
		{
			name:       "failure - empty statuses",
			vmStatuses: []string{},
			vmName:     "master-0",
			wantErr:    "expected 2 statuses for VM master-0, but found 0: ",
		},
		{
			name: "failure - wrong provisioning state",
			vmStatuses: []string{
				"ProvisioningState/failed",
				"PowerState/running",
			},
			vmName:  "master-0",
			wantErr: "found unexpected statuses for VM master-0: ProvisioningState/failed",
		},
		{
			name: "failure - wrong power state",
			vmStatuses: []string{
				"ProvisioningState/succeeded",
				"PowerState/stopped",
			},
			vmName:  "master-0",
			wantErr: "found unexpected statuses for VM master-0: PowerState/stopped",
		},
		{
			name: "failure - both statuses are wrong",
			vmStatuses: []string{
				"ProvisioningState/failed",
				"PowerState/deallocated",
			},
			vmName:  "master-0",
			wantErr: "found unexpected statuses for VM master-0: ProvisioningState/failed, PowerState/deallocated",
		},
		{
			name: "failure - VM deallocating",
			vmStatuses: []string{
				"ProvisioningState/succeeded",
				"PowerState/deallocating",
			},
			vmName:  "master-1",
			wantErr: "found unexpected statuses for VM master-1: PowerState/deallocating",
		},
		{
			name: "failure - VM starting",
			vmStatuses: []string{
				"ProvisioningState/succeeded",
				"PowerState/starting",
			},
			vmName:  "master-2",
			wantErr: "found unexpected statuses for VM master-2: PowerState/starting",
		},
		{
			name: "failure - completely wrong statuses",
			vmStatuses: []string{
				"CustomState/value1",
				"CustomState/value2",
			},
			vmName:  "master-0",
			wantErr: "found unexpected statuses for VM master-0: CustomState/value1, CustomState/value2",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVMPowerState(log, tt.vmStatuses, tt.vmName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

// TestGetAzureVMs tests the getAzureVMs function with various scenarios
func TestGetAzureVMs(t *testing.T) {
	_, log := testlog.New()
	ctx := context.Background()

	// Helper to create a simple machines map with master nodes
	createMachinesMap := func(names ...string) map[string]machineValidationData {
		machines := make(map[string]machineValidationData)
		for _, name := range names {
			machines[name] = machineValidationData{
				labelZone:         "1",
				specZone:          "1",
				size:              "Standard_D8s_v3",
				phase:             "Running",
				labelInstanceType: "Standard_D8s_v3",
			}
		}
		return machines
	}

	for _, tt := range []struct {
		name      string
		machines  map[string]machineValidationData
		mocks     func(*mock_adminactions.MockAzureActions)
		wantErr   string
		wantCount int // expected number of VMs returned
	}{
		{
			name:     "success - 3 master VMs in 3 zones",
			machines: createMachinesMap("master-0", "master-1", "master-2"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				zones0 := []string{"1"}
				zones1 := []string{"2"}
				zones2 := []string{"3"}

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones0,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-1", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones1,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-2", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones2,
					}, nil)
			},
			wantCount: 3, // All 3 master VMs
		},
		{
			name:     "failure - VM not found in Azure",
			machines: createMachinesMap("master-0"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{}, fmt.Errorf("VM not found"))
			},
			wantErr: "VM not found",
		},
		{
			name:     "failure - VM with no zones",
			machines: createMachinesMap("master-0"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				emptyZones := []string{}
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &emptyZones,
					}, nil)
			},
			wantErr: "azure VM master-0 has no availability zone configured",
		},
		{
			name:     "failure - VM with nil zones",
			machines: createMachinesMap("master-0"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: nil,
					}, nil)
			},
			wantErr: "azure VM master-0 has no availability zone configured",
		},
		{
			name:     "failure - wrong zone distribution (only 2 zones)",
			machines: createMachinesMap("master-0", "master-1", "master-2"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				zones0 := []string{"1"}
				zones1 := []string{"2"}
				zones2 := []string{"1"} // Duplicate zone

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones0,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-1", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones1,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-2", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones2,
					}, nil)
			},
			wantErr: "items must be spread across 3 different zones, found 2 zone(s)",
		},
		{
			name:     "handles nil InstanceView gracefully",
			machines: createMachinesMap("master-0", "master-1", "master-2"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				zones0 := []string{"1"}
				zones1 := []string{"2"}
				zones2 := []string{"3"}

				// master-0 has nil InstanceView
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: nil, // nil InstanceView
						},
						Zones: &zones0,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-1", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones1,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-2", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones2,
					}, nil)
			},
			wantErr: "expected 2 statuses for VM master-0, but found 0: ",
		},
		{
			name:     "handles nil InstanceView.Statuses gracefully",
			machines: createMachinesMap("master-0", "master-1", "master-2"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				zones0 := []string{"1"}
				zones1 := []string{"2"}
				zones2 := []string{"3"}

				// master-0 has nil Statuses
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: nil, // nil Statuses
							},
						},
						Zones: &zones0,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-1", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones1,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-2", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones2,
					}, nil)
			},
			wantErr: "expected 2 statuses for VM master-0, but found 0: ",
		},
		{
			name:     "handles nil HardwareProfile gracefully",
			machines: createMachinesMap("master-0", "master-1", "master-2"),
			mocks: func(a *mock_adminactions.MockAzureActions) {
				zones0 := []string{"1"}
				zones1 := []string{"2"}
				zones2 := []string{"3"}

				// master-0 has nil HardwareProfile
				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-0", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: nil, // nil HardwareProfile
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones0,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-1", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones1,
					}, nil)

				a.EXPECT().GetVirtualMachine(ctx, "test-cluster", "master-2", mgmtcompute.InstanceView).Return(
					mgmtcompute.VirtualMachine{
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							HardwareProfile: &mgmtcompute.HardwareProfile{
								VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD8sV3,
							},
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("ProvisioningState/succeeded")},
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
						Zones: &zones2,
					}, nil)
			},
			wantCount: 3, // All 3 VMs are added, but master-0 will have empty vmSize
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			azureAction := mock_adminactions.NewMockAzureActions(ctrl)
			tt.mocks(azureAction)

			vms, err := getAzureVMs(log, ctx, azureAction, "/subscriptions/test/resourceGroups/test-cluster", tt.machines)

			if tt.wantErr != "" {
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(vms) != tt.wantCount {
					t.Errorf("expected %d VMs, got %d", tt.wantCount, len(vms))
				}
			}
		})
	}
}

// TestValidateClusterNodes tests the validateClusterNodes function
func TestValidateClusterNodes(t *testing.T) {
	_, log := testlog.New()
	ctx := context.Background()

	// Helper to create a node object
	// isControlPlane: true for control plane nodes (sets master label with empty value), false for worker nodes
	// nodeInstanceType and betaInstanceType can be empty strings to omit those labels
	createNode := func(name string, isControlPlane bool, unschedulable bool, ready bool, nodeInstanceType, betaInstanceType string) corev1.Node {
		labels := map[string]string{}
		if isControlPlane {
			labels["node-role.kubernetes.io/master"] = ""
		}
		if nodeInstanceType != "" {
			labels["node.kubernetes.io/instance-type"] = nodeInstanceType
		}
		if betaInstanceType != "" {
			labels["beta.kubernetes.io/instance-type"] = betaInstanceType
		}

		conditions := []corev1.NodeCondition{}
		if ready {
			conditions = append(conditions, corev1.NodeCondition{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			})
		} else {
			conditions = append(conditions, corev1.NodeCondition{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionFalse,
			})
		}

		return corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: corev1.NodeSpec{
				Unschedulable: unschedulable,
			},
			Status: corev1.NodeStatus{
				Conditions: conditions,
			},
		}
	}

	// Helper to encode node list to bytes
	encodeNodeList := func(nodes ...corev1.Node) []byte {
		nodeList := &corev1.NodeList{
			Items: nodes,
		}
		b, _ := json.Marshal(nodeList)
		return b
	}

	for _, tt := range []struct {
		name      string
		mocks     func(*mock_adminactions.MockKubeActions)
		wantErr   string
		wantCount int
	}{
		{
			name: "success - 3 control plane nodes, all ready and schedulable",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - filters worker nodes",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("worker-0", false, false, true, "Standard_D8s_v3", "Standard_D8s_v3"), // Worker node (role != "")
					createNode("worker-1", false, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantCount: 3,
		},
		{
			name: "failure - node is unschedulable",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, true, true, "Standard_D8s_v3", "Standard_D8s_v3"), // Unschedulable
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "node master-1 is unschedulable",
		},
		{
			name: "failure - node is not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, false, "Standard_D8s_v3", "Standard_D8s_v3"), // Not ready
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "node master-1 is not ready",
		},
		{
			name: "failure - multiple nodes unschedulable",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, true, true, "Standard_D8s_v3", "Standard_D8s_v3"), // Unschedulable
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-2", true, true, true, "Standard_D8s_v3", "Standard_D8s_v3"), // Unschedulable
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "master-0 is unschedulable",
		},
		{
			name: "failure - multiple nodes not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, false, "Standard_D8s_v3", "Standard_D8s_v3"), // Not ready
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-2", true, false, false, "Standard_D8s_v3", "Standard_D8s_v3"), // Not ready
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "master-0 is not ready",
		},
		{
			name: "failure - node both unschedulable and not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, true, false, "Standard_D8s_v3", "Standard_D8s_v3"), // Both issues
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "master-1 is unschedulable",
		},
		{
			name: "failure - only 2 control plane nodes",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "expected 3 control plane nodes, found 2",
		},
		{
			name: "failure - 4 control plane nodes",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-3", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "expected 3 control plane nodes, found 4",
		},
		{
			name: "failure - 0 control plane nodes",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("worker-0", false, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("worker-1", false, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "expected 3 control plane nodes, found 0",
		},
		{
			name: "failure - combination of issues",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, true, true, "Standard_D8s_v3", "Standard_D8s_v3"),   // Unschedulable
					createNode("master-1", true, false, false, "Standard_D8s_v3", "Standard_D8s_v3"), // Not ready
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "master-0 is unschedulable",
		},
		{
			name: "failure - node instance-type labels mismatch",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D8s_v3", "Standard_D16s_v3"), // Mismatch
					createNode("master-2", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantErr: "node master-1 has a mismatch between labels",
		},
		{
			name: "success - nodes with matching instance-type labels",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				nodes := encodeNodeList(
					createNode("master-0", true, false, true, "Standard_D8s_v3", "Standard_D8s_v3"),
					createNode("master-1", true, false, true, "Standard_D16s_v3", "Standard_D16s_v3"),
					createNode("master-2", true, false, true, "Standard_D32s_v3", "Standard_D32s_v3"),
				)
				k.EXPECT().KubeList(ctx, "Node", "").Return(nodes, nil)
			},
			wantCount: 3,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			kubeActions := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(kubeActions)

			nodes, err := validateClusterNodes(log, ctx, kubeActions)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error to contain %q, got: %s", tt.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(nodes) != tt.wantCount {
					t.Errorf("expected %d nodes, got %d", tt.wantCount, len(nodes))
				}
			}
		})
	}
}

// TestValidateClusterMachinesAndNodes tests the validateClusterMachinesAndNodes function
func TestValidateClusterMachinesAndNodes(t *testing.T) {
	_, log := testlog.New()

	for _, tt := range []struct {
		name           string
		ocMachines     map[string]machineValidationData
		ocNodes        map[string]nodeValidationData
		wantErrStrings []string
	}{
		{
			name: "valid - all machines match nodes",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			ocNodes: map[string]nodeValidationData{
				"master-0": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
				"master-1": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
				"master-2": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
			},
		},
		{
			name: "invalid - machine not found in nodes",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			ocNodes: map[string]nodeValidationData{
				"master-0": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
				"master-1": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{"machine master-2 not found in cluster nodes"},
		},
		{
			name: "invalid - instance-type mismatch",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			ocNodes: map[string]nodeValidationData{
				"master-0": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
				"master-1": {nodeInstanceType: "Standard_D16s_v3", betaInstanceType: "Standard_D16s_v3"}, // Wrong size
				"master-2": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{"machine master-1 has size Standard_D8s_v3 in its spec, however node has instance-type Standard_D16s_v3"},
		},
		{
			name: "invalid - multiple errors collected",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			ocNodes: map[string]nodeValidationData{
				"master-0": {nodeInstanceType: "Standard_D16s_v3", betaInstanceType: "Standard_D16s_v3"}, // Wrong size
				"master-1": {nodeInstanceType: "Standard_D8s_v3", betaInstanceType: "Standard_D8s_v3"},
			},
			wantErrStrings: []string{
				"machine master-0 has size Standard_D8s_v3 in its spec, however node has instance-type Standard_D16s_v3",
				"machine master-2 not found in cluster nodes",
			},
		},
		{
			name: "invalid - all machines have mismatched nodes",
			ocMachines: map[string]machineValidationData{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			ocNodes: map[string]nodeValidationData{
				"master-0": {nodeInstanceType: "Standard_D16s_v3", betaInstanceType: "Standard_D16s_v3"},
				"master-1": {nodeInstanceType: "Standard_D32s_v3", betaInstanceType: "Standard_D32s_v3"},
				"master-2": {nodeInstanceType: "Standard_D64s_v3", betaInstanceType: "Standard_D64s_v3"},
			},
			wantErrStrings: []string{
				"master-0 has size Standard_D8s_v3",
				"master-1 has size Standard_D8s_v3",
				"master-2 has size Standard_D8s_v3",
			},
		},
		{
			name:       "valid - empty maps",
			ocMachines: map[string]machineValidationData{},
			ocNodes:    map[string]nodeValidationData{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterMachinesAndNodes(log, tt.ocMachines, tt.ocNodes)

			if len(tt.wantErrStrings) > 0 {
				if err == nil {
					t.Fatalf("expected error with messages %v, got nil", tt.wantErrStrings)
				}
				errStr := err.Error()
				for _, wantErrString := range tt.wantErrStrings {
					if !strings.Contains(errStr, wantErrString) {
						t.Errorf("expected error to contain %q, but got: %s", wantErrString, errStr)
					}
				}
			} else {
				utilerror.AssertErrorMessage(t, err, "")
			}
		})
	}
}
