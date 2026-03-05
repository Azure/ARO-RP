package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

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
				"item3": "1",
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
		ocMachines     map[string]machineBasics
		azureVMs       map[string]azureVMBasics
		wantErr        string
		wantErrStrings []string // For tests with multiple errors, check each string is present
	}{
		{
			name: "valid - all machines match Azure VMs",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D8s_v3", zone: "1"},
				"master-1": {status: []string{"PowerState/running"}, vmSize: "Standard_D8s_v3", zone: "2"},
				"master-2": {status: []string{"PowerState/running"}, vmSize: "Standard_D8s_v3", zone: "3"},
			},
		},
		{
			name: "invalid - machine not found in Azure",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D8s_v3", zone: "1"},
			},
			wantErr: "machine master-1 not found in Azure resources",
		},
		{
			name: "invalid - zone mismatch",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D8s_v3", zone: "2"},
			},
			wantErr: "machine master-0 has zone 1 in its spec, however Azure VM is running in zone 2",
		},
		{
			name: "invalid - size mismatch",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D16s_v3", zone: "1"},
			},
			wantErr: "machine master-0 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM",
		},
		{
			name: "invalid - multiple errors collected",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
				"master-1": {labelZone: "2", specZone: "2", size: "Standard_D8s_v3"},
				"master-2": {labelZone: "3", specZone: "3", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D16s_v3", zone: "2"},
				"master-1": {status: []string{"PowerState/running"}, vmSize: "Standard_D4s_v3", zone: "2"},
			},
			// errors.Join() order depends on map iteration - check all error strings are present
			wantErrStrings: []string{
				"machine master-0 has zone 1 in its spec, however Azure VM is running in zone 2",
				"machine master-0 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM",
				"machine master-1 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D4s_v3 VM",
				"machine master-2 not found in Azure resources",
			},
		},
		{
			name: "invalid - zone and size mismatch for same machine",
			ocMachines: map[string]machineBasics{
				"master-0": {labelZone: "1", specZone: "1", size: "Standard_D8s_v3"},
			},
			azureVMs: map[string]azureVMBasics{
				"master-0": {status: []string{"PowerState/running"}, vmSize: "Standard_D16s_v3", zone: "2"},
			},
			wantErr: "machine master-0 has zone 1 in its spec, however Azure VM is running in zone 2\nmachine master-0 has size Standard_D8s_v3 in its spec, however Azure VM is running a Standard_D16s_v3 VM",
		},
		{
			name:       "valid - empty maps",
			ocMachines: map[string]machineBasics{},
			azureVMs:   map[string]azureVMBasics{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterMachinesAndVMs(log, tt.ocMachines, tt.azureVMs)

			// For tests with multiple errors, check each error string is present
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
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			}
		})
	}
}

// TestGetClusterMachines tests the getClusterMachines function with various scenarios
func TestGetClusterMachines(t *testing.T) {
	_, log := testlog.New()
	ctx := context.Background()

	// Helper to create a machine object
	// phase can be empty string to create a machine with nil Phase
	createMachine := func(name, role, labelZone, specZone, vmSize, phase string) machinev1beta1.Machine {
		providerSpec := &machinev1beta1.AzureMachineProviderSpec{
			Zone:   &specZone,
			VMSize: vmSize,
		}
		providerSpecRaw, _ := json.Marshal(providerSpec)

		labels := map[string]string{
			"machine.openshift.io/zone": labelZone,
		}
		if role != "" {
			labels["machine.openshift.io/cluster-api-machine-role"] = role
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
			name: "success - 3 master nodes with matching zones",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - filters non-master machines",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
					createMachine("worker-0", "worker", "1", "1", "Standard_D4s_v3", "Running"),
					createMachine("worker-1", "worker", "2", "2", "Standard_D4s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters
		},
		{
			name: "failure - zone mismatch between label and spec",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "2", "Standard_D8s_v3", "Running"), // Label zone 1, spec zone 2
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "machine master-0 has a mismatch between label zone 1 and spec zone 2. These values should match",
		},
		{
			name: "failure - multiple zone mismatches",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "2", "Standard_D8s_v3", "Running"), // Mismatch
					createMachine("master-1", "master", "2", "3", "Standard_D8s_v3", "Running"), // Mismatch
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"), // OK
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			// Should contain both errors since we collect all validation errors
			wantErr: "master-0 has a mismatch",
		},
		{
			name: "failure - only 2 master nodes (zone distribution fails)",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "expected 3 items, got 2",
		},
		{
			name: "failure - 3 masters but only 2 zones",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "1", "1", "Standard_D8s_v3", "Running"), // Duplicate zone
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "items must be spread across 3 different zones, found 2 zone(s)",
		},
		{
			name: "success - filters machines without role label",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
					createMachine("master-3", "", "1", "1", "Standard_D8s_v3", "Running"), // No role label
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters with proper role label
		},
		{
			name: "success - filters machines named master- but with wrong role",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
					createMachine("master-infra-0", "infra", "1", "1", "Standard_D8s_v3", "Running"), // Has "master" in name but wrong role
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantCount: 3, // Only masters with role=master
		},
		{
			name: "failure - machine with nil phase",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", ""), // nil phase
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "machine master-1 status phase is not Running, current phase is nil",
		},
		{
			name: "failure - machine with Provisioning phase",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Provisioning"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "machine master-1 status phase is not Running, current phase is Provisioning",
		},
		{
			name: "failure - machine with Failed phase",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Running"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", "Running"),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Failed"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "machine master-2 status phase is not Running, current phase is Failed",
		},
		{
			name: "failure - multiple machines with wrong phase",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				machines := encodeMachineList(
					createMachine("master-0", "master", "1", "1", "Standard_D8s_v3", "Deleting"),
					createMachine("master-1", "master", "2", "2", "Standard_D8s_v3", ""),
					createMachine("master-2", "master", "3", "3", "Standard_D8s_v3", "Running"),
				)
				k.EXPECT().KubeList(ctx, "Machine", machineNamespace).Return(machines, nil)
			},
			wantErr: "master-0 status phase is not Running",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			kubeActions := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(kubeActions)

			machines, err := getClusterMachines(log, ctx, kubeActions)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
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

	for _, tt := range []struct {
		name      string
		mocks     func(*mock_adminactions.MockAzureActions)
		wantErr   string
		wantCount int // expected number of VMs returned
	}{
		{
			name: "success - 3 master VMs in 3 zones",
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GroupResourceList(ctx).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name: pointerutils.ToPtr("master-0"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: pointerutils.ToPtr("master-1"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: pointerutils.ToPtr("master-2"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: pointerutils.ToPtr("worker-0"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
				}, nil)

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
			name: "failure - VM with no zones",
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GroupResourceList(ctx).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name: pointerutils.ToPtr("master-0"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
				}, nil)

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
			name: "failure - VM with nil zones",
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GroupResourceList(ctx).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name: pointerutils.ToPtr("master-0"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
				}, nil)

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
			name: "failure - wrong zone distribution (only 2 zones)",
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GroupResourceList(ctx).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name: pointerutils.ToPtr("master-0"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: pointerutils.ToPtr("master-1"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: pointerutils.ToPtr("master-2"),
						Type: pointerutils.ToPtr("Microsoft.Compute/virtualMachines"),
					},
				}, nil)

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
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			azureAction := mock_adminactions.NewMockAzureActions(ctrl)
			tt.mocks(azureAction)

			vms, err := getAzureVMs(log, ctx, azureAction, "/subscriptions/test/resourceGroups/test-cluster")

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
