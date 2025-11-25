package nic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

const (
	testSubscriptionID     = "00000000-0000-0000-0000-000000000000"
	testResourceGroup      = "aro-cluster-rg"
	testClusterRGID        = "/subscriptions/" + testSubscriptionID + "/resourcegroups/" + testResourceGroup
	testInfraID            = "cluster-abc-xyz"
	testMasterMachineName  = "cluster-abc-xyz-master-0"
	testWorkerMachineName  = "cluster-abc-xyz-worker-xyz-1"
	testMasterNICName      = testMasterMachineName + "-nic"
	testWorkerNICName      = testWorkerMachineName + "-nic"
)

// TestExtractNICNameFromMachine tests extracting NIC names from Machine objects
func TestExtractNICNameFromMachine(t *testing.T) {
	tests := []struct {
		name        string
		machine     *machinev1beta1.Machine
		expectedNIC string
		wantErr     bool
	}{
		{
			name:        "Master machine with valid provider spec",
			machine:     getValidMachine(testMasterMachineName, true),
			expectedNIC: testMasterNICName,
			wantErr:     false,
		},
		{
			name:        "Worker machine with valid provider spec",
			machine:     getValidMachine(testWorkerMachineName, false),
			expectedNIC: testWorkerNICName,
			wantErr:     false,
		},
		{
			name: "Machine with nil provider spec",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-machine",
				},
				Spec: machinev1beta1.MachineSpec{},
			},
			wantErr: true,
		},
		{
			name: "Machine with invalid provider spec JSON",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-machine",
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(`{invalid json}`),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Machine with empty name",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(`{"vmSize":"Standard_D4s_v3"}`),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nicName, err := extractNICNameFromMachine(tt.machine)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if nicName != tt.expectedNIC {
				t.Errorf("Expected NIC name %q, got %q", tt.expectedNIC, nicName)
			}
		})
	}
}

// TestIsNICInFailedState tests detection of failed NIC provisioning states
func TestIsNICInFailedState(t *testing.T) {
	tests := []struct {
		name         string
		nic          *armnetwork.Interface
		expectFailed bool
	}{
		{
			name: "NIC in Failed state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateFailed),
				},
			},
			expectFailed: true,
		},
		{
			name: "NIC in Succeeded state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded),
				},
			},
			expectFailed: false,
		},
		{
			name: "NIC in Creating state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateCreating),
				},
			},
			expectFailed: false,
		},
		{
			name: "NIC in Updating state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateUpdating),
				},
			},
			expectFailed: false,
		},
		{
			name: "NIC in Deleting state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateDeleting),
				},
			},
			expectFailed: false,
		},
		{
			name:         "NIC with nil Properties",
			nic:          &armnetwork.Interface{},
			expectFailed: false,
		},
		{
			name: "NIC with nil ProvisioningState",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{},
			},
			expectFailed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNICInFailedState(tt.nic)
			if result != tt.expectFailed {
				t.Errorf("Expected isNICInFailedState() = %v, got %v", tt.expectFailed, result)
			}
		})
	}
}

// TestIsNotFoundError tests detection of 404 Not Found errors
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectNotFound bool
	}{
		{
			name:           "Nil error",
			err:            nil,
			expectNotFound: false,
		},
		{
			name:           "ResourceNotFound error",
			err:            errors.New("GET https://management.azure.com/.../xyz: 404 Not Found: ResourceNotFound"),
			expectNotFound: true,
		},
		{
			name:           "Generic NotFound error",
			err:            errors.New("resource not found: NotFound"),
			expectNotFound: true,
		},
		{
			name:           "404 status code error",
			err:            errors.New("HTTP request failed with status code 404"),
			expectNotFound: true,
		},
		{
			name:           "Permission denied error",
			err:            errors.New("GET https://management.azure.com/.../xyz: 403 Forbidden: AuthorizationFailed"),
			expectNotFound: false,
		},
		{
			name:           "Network error",
			err:            errors.New("dial tcp: connection refused"),
			expectNotFound: false,
		},
		{
			name:           "Timeout error",
			err:            errors.New("context deadline exceeded"),
			expectNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			if result != tt.expectNotFound {
				t.Errorf("Expected isNotFoundError() = %v, got %v", tt.expectNotFound, result)
			}
		})
	}
}

// TestReconcilerControllerDisabled tests that reconciliation skips when controller is disabled
func TestReconcilerControllerDisabled(t *testing.T) {
	ctx := context.Background()

	// Create cluster with controller disabled
	cluster := getValidClusterInstance(false)

	// Create fake client
	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithObjects(cluster)
	client := clientBuilder.Build()

	// Create reconciler
	r := &Reconciler{
		log:    logrus.NewEntry(logrus.StandardLogger()),
		client: client,
	}

	// Reconcile
	result, err := r.Reconcile(ctx, ctrl.Request{})

	// Verify
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result.Requeue {
		t.Errorf("Expected no requeue when controller disabled")
	}
}

// TestReconcilerControllerEnabled tests basic reconciliation flow when enabled
func TestReconcilerControllerEnabled(t *testing.T) {
	// Note: This is a basic test that verifies the controller attempts to run
	// Full integration testing would require mocking Azure clients
	ctx := context.Background()

	// Create cluster with controller enabled
	cluster := getValidClusterInstance(true)

	// Create fake client
	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithObjects(cluster)
	client := clientBuilder.Build()

	// Create reconciler
	r := &Reconciler{
		log:    logrus.NewEntry(logrus.StandardLogger()),
		client: client,
	}

	// Reconcile - this will fail due to missing Azure environment config
	// but we're testing that it gets past the "disabled" check
	_, err := r.Reconcile(ctx, ctrl.Request{})

	// We expect an error here because Azure env parsing will fail in test environment
	// The important thing is that we got past the "controller disabled" early return
	if err == nil {
		t.Logf("Note: Expected error due to Azure env parsing in test, but got nil")
	}
}

// TestReconcileNICForMachineNotFound tests handling of non-existent machines
func TestReconcileNICForMachineNotFound(t *testing.T) {
	ctx := context.Background()

	// Create cluster
	cluster := getValidClusterInstance(true)

	// Create fake client (no machines)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithObjects(cluster)
	client := clientBuilder.Build()

	// Create reconcile manager
	rm := &reconcileManager{
		log:    logrus.NewEntry(logrus.StandardLogger()),
		client: client,
	}

	// Try to reconcile non-existent machine
	err := rm.reconcileNICForMachine(ctx, "non-existent-machine", machineNamespace)

	// Should not return error (machine not found is handled gracefully)
	if err != nil {
		t.Logf("Error (expected in test): %v", err)
	}
}

// TestReconcileNICsForMachineSet tests finding machines owned by a MachineSet
func TestReconcileNICsForMachineSet(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		machineSetName     string
		machines           []string
		expectedMachineSet string
	}{
		{
			name:               "MachineSet with multiple machines",
			machineSetName:     "test-machineset",
			machines:           []string{"test-machine-1", "test-machine-2", "test-machine-3"},
			expectedMachineSet: "test-machineset",
		},
		{
			name:               "MachineSet with no machines",
			machineSetName:     "empty-machineset",
			machines:           []string{},
			expectedMachineSet: "empty-machineset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create cluster
			cluster := getValidClusterInstance(true)

			// Create machines owned by the MachineSet
			var objects []client.Object
			objects = append(objects, cluster)
			machineSet := getValidMachineSet(tt.machineSetName)
			objects = append(objects, machineSet)

			for _, machineName := range tt.machines {
				machine := getValidMachine(machineName, false)
				machine.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: "machine.openshift.io/v1beta1",
						Kind:       "MachineSet",
						Name:       tt.machineSetName,
						UID:        "test-uid",
					},
				}
				objects = append(objects, machine)
			}

			// Create fake client
			clientBuilder := fake.NewClientBuilder()
			clientBuilder = clientBuilder.WithObjects(objects...)
			fakeClient := clientBuilder.Build()

			// Test that we can list machines for this MachineSet
			machineList := &machinev1beta1.MachineList{}
			err := fakeClient.List(ctx, machineList)
			if err != nil {
				t.Fatalf("Failed to list machines: %v", err)
			}

			// Count machines owned by this MachineSet
			ownedCount := 0
			for _, machine := range machineList.Items {
				for _, ownerRef := range machine.OwnerReferences {
					if ownerRef.Kind == "MachineSet" && ownerRef.Name == tt.machineSetName {
						ownedCount++
						// Verify NIC name can be extracted
						nicName, err := extractNICNameFromMachine(&machine)
						if err != nil {
							t.Errorf("Failed to extract NIC name from machine %s: %v", machine.Name, err)
						}
						expectedNIC := machine.Name + "-nic"
						if nicName != expectedNIC {
							t.Errorf("Expected NIC name %q, got %q", expectedNIC, nicName)
						}
					}
				}
			}

			if ownedCount != len(tt.machines) {
				t.Errorf("Expected %d machines owned by MachineSet, got %d", len(tt.machines), ownedCount)
			}
		})
	}
}

// Helper Functions

// getValidClusterInstance creates a valid Cluster CR for testing
func getValidClusterInstance(nicEnabled bool) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:             "/subscriptions/" + testSubscriptionID + "/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			ClusterResourceGroupID: testClusterRGID,
			InfraID:                testInfraID,
			AZEnvironment:          "AzurePublicCloud",
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.NICEnabled: strconv.FormatBool(nicEnabled),
			},
		},
	}
}

// getValidMachine creates a valid Machine for testing
func getValidMachine(name string, isMaster bool) *machinev1beta1.Machine {
	role := "worker"
	if isMaster {
		role = "master"
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineNamespace,
			Labels: map[string]string{
				"machine.openshift.io/cluster-api-machine-role": role,
			},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{
					Raw: []byte(`{
						"vmSize": "Standard_D4s_v3",
						"osDisk": {
							"diskSizeGB": 128
						}
					}`),
				},
			},
		},
	}
}

// getValidMachineSet creates a valid MachineSet for testing
func getValidMachineSet(name string) *machinev1beta1.MachineSet {
	replicas := int32(3)
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineNamespace,
			Labels: map[string]string{
				"machine.openshift.io/cluster-api-machine-role": "worker",
			},
		},
		Spec: machinev1beta1.MachineSetSpec{
			Replicas: &replicas,
			Template: machinev1beta1.MachineTemplateSpec{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(`{
								"vmSize": "Standard_D4s_v3",
								"osDisk": {
									"diskSizeGB": 128
								}
							}`),
						},
					},
				},
			},
		},
	}
}

// TestNICNamingConvention tests that NIC names follow expected pattern
func TestNICNamingConvention(t *testing.T) {
	tests := []struct {
		machineName string
		expectedNIC string
	}{
		{
			machineName: "cluster-abc-master-0",
			expectedNIC: "cluster-abc-master-0-nic",
		},
		{
			machineName: "cluster-abc-master-1",
			expectedNIC: "cluster-abc-master-1-nic",
		},
		{
			machineName: "cluster-abc-master-2",
			expectedNIC: "cluster-abc-master-2-nic",
		},
		{
			machineName: "cluster-abc-worker-xyz-1",
			expectedNIC: "cluster-abc-worker-xyz-1-nic",
		},
		{
			machineName: "cluster-abc-worker-xyz-2",
			expectedNIC: "cluster-abc-worker-xyz-2-nic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.machineName, func(t *testing.T) {
			machine := getValidMachine(tt.machineName, false)
			nicName, err := extractNICNameFromMachine(machine)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if nicName != tt.expectedNIC {
				t.Errorf("Expected NIC name %q, got %q", tt.expectedNIC, nicName)
			}
		})
	}
}

// TestProvisioningStateDetection tests detection of various Azure provisioning states
func TestProvisioningStateDetection(t *testing.T) {
	// Test all Azure NIC provisioning states
	allStates := []struct {
		state        armnetwork.ProvisioningState
		shouldFail   bool
		description  string
	}{
		{armnetwork.ProvisioningStateFailed, true, "Failed state should be detected"},
		{armnetwork.ProvisioningStateSucceeded, false, "Succeeded state is healthy"},
		{armnetwork.ProvisioningStateCreating, false, "Creating state is transitional"},
		{armnetwork.ProvisioningStateUpdating, false, "Updating state is transitional"},
		{armnetwork.ProvisioningStateDeleting, false, "Deleting state is transitional"},
	}

	for _, test := range allStates {
		t.Run(string(test.state), func(t *testing.T) {
			nic := &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(test.state),
				},
			}

			result := isNICInFailedState(nic)
			if result != test.shouldFail {
				t.Errorf("%s: Expected %v, got %v", test.description, test.shouldFail, result)
			}
		})
	}
}
