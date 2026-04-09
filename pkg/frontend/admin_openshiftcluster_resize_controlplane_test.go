package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func masterMachineListJSON(machines ...machinev1beta1.Machine) []byte {
	list := &machinev1beta1.MachineList{Items: machines}
	b, _ := json.Marshal(list)
	return b
}

func masterMachine(name, vmSize, phase string) machinev1beta1.Machine {
	providerSpec := &machinev1beta1.AzureMachineProviderSpec{
		Zone:   strPtr("1"),
		VMSize: vmSize,
	}
	raw, _ := json.Marshal(providerSpec)

	m := machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				machineLabelClusterAPIRole: machineRoleMaster,
				machineLabelZone:           "1",
				machineLabelInstanceType:   vmSize,
			},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{Raw: raw},
			},
		},
	}
	if phase != "" {
		m.Status.Phase = &phase
	}
	return m
}

func strPtr(s string) *string { return &s }

func cpmsJSON(state string) []byte {
	obj := map[string]interface{}{
		"apiVersion": "machine.openshift.io/v1",
		"kind":       "ControlPlaneMachineSet",
		"metadata":   map[string]interface{}{"name": "cluster", "namespace": machineNamespace},
		"spec":       map[string]interface{}{"state": state},
	}
	b, _ := json.Marshal(obj)
	return b
}

func nodeJSON(name string, ready bool) []byte {
	return nodeJSONWithSchedulability(name, ready, false)
}

func nodeJSONWithSchedulability(name string, ready, unschedulable bool) []byte {
	status := "False"
	if ready {
		status = "True"
	}
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata":   map[string]interface{}{"name": name},
		"spec": map[string]interface{}{
			"unschedulable": unschedulable,
		},
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Ready", "status": status},
			},
		},
	}
	b, _ := json.Marshal(obj)
	return b
}

func machineJSON(name, vmSize string) []byte {
	obj := map[string]interface{}{
		"apiVersion": "machine.openshift.io/v1beta1",
		"kind":       "Machine",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         machineNamespace,
			"creationTimestamp": "2024-01-01T00:00:00Z",
			"labels":            map[string]interface{}{machineLabelInstanceType: vmSize},
		},
		"spec": map[string]interface{}{
			"providerSpec": map[string]interface{}{
				"value": map[string]interface{}{
					"vmSize": vmSize,
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
					},
				},
			},
		},
	}
	b, _ := json.Marshal(obj)
	return b
}

func TestCheckCPMSNotActive(t *testing.T) {
	ctx := context.Background()

	cpmsGR := schema.GroupResource{Group: "machine.openshift.io", Resource: "controlplanemachinesets"}

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "CPMS not found - safe to proceed",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(nil, kerrors.NewNotFound(cpmsGR, "cluster"))
			},
		},
		{
			name: "CPMS inactive - safe to proceed",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(cpmsJSON("Inactive"), nil)
			},
		},
		{
			name: "CPMS active - blocked",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(cpmsJSON("Active"), nil)
			},
			wantErr: "409: RequestNotAllowed: : ControlPlaneMachineSet is currently Active. Deactivate CPMS before running this operation.",
		},
		{
			name: "CPMS with empty state - safe to proceed",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(cpmsJSON(""), nil)
			},
		},
		{
			name: "KubeGet returns non-NotFound error - fails closed",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(nil, errors.New("connection refused"))
			},
			wantErr: "500: InternalServerError: : failed to check ControlPlaneMachineSet state: connection refused",
		},
		{
			name: "CPMS returns invalid JSON - fails closed",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return([]byte("not-json"), nil)
			},
			wantErr: "500: InternalServerError: : failed to parse ControlPlaneMachineSet object: invalid character 'o' in literal null (expecting 'u')",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(k)

			err := checkCPMSNotActive(ctx, k)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestIsNodeReady(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name      string
		mocks     func(*mock_adminactions.MockKubeActions)
		wantReady bool
		wantErr   string
	}{
		{
			name: "node is ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", true), nil)
			},
			wantReady: true,
		},
		{
			name: "node is not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", false), nil)
			},
			wantReady: false,
		},
		{
			name: "node not found",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nil, errors.New("not found"))
			},
			wantReady: false,
			wantErr:   "not found",
		},
		{
			name: "node payload invalid JSON",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return([]byte(`{invalid`), nil)
			},
			wantReady: false,
			wantErr:   "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "node without conditions is treated as not ready",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return([]byte(`{"apiVersion":"v1","kind":"Node","metadata":{"name":"master-0"},"status":{}}`), nil)
			},
			wantReady: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(k)

			ready, err := isNodeReady(ctx, k, "master-0")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if ready != tt.wantReady {
				t.Errorf("got ready=%v, want %v", ready, tt.wantReady)
			}
		})
	}
}

func TestResizeControlPlane(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	running := "Running"
	desiredSize := "Standard_D16s_v5"

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions, *mock_adminactions.MockAzureActions)
		wantErr string
	}{
		{
			name: "all nodes already at desired size - no-op",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				machines := masterMachineListJSON(
					masterMachine("master-0", desiredSize, running),
					masterMachine("master-1", desiredSize, running),
					masterMachine("master-2", desiredSize, running),
				)
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(machines, nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").
					Return(nodeJSON("master-2", true), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").
					Return(nodeJSON("master-1", true), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", true), nil)
			},
		},
		{
			name: "single node resize - full sequence",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				machines := masterMachineListJSON(
					masterMachine("master-0", desiredSize, running),
					masterMachine("master-1", desiredSize, running),
					masterMachine("master-2", "Standard_D8s_v3", running),
				)
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(machines, nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").
						Return(nodeJSON("master-2", true), nil),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").
						Return(nodeJSON("master-1", true), nil),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-2", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-2").Return(nil),
					a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
					a.EXPECT().VMResize(gomock.Any(), "master-2", desiredSize).Return(nil),
					a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").
						Return(nodeJSON("master-2", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-2", false).Return(nil),
					k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
						Return(machineJSON("master-2", "Standard_D8s_v3"), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").
						Return(nodeJSON("master-2", true), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
				)
			},
		},
		{
			name: "drain fails",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").
						Return(errors.New("could not drain node after 3 retries: drain error")),
				)
			},
			wantErr: "failed to resize node master-0: draining node: could not drain node after 3 retries: drain error",
		},
		{
			name: "stop VM fails",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
					a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).
						Return(errors.New("Azure capacity error")),
				)
			},
			wantErr: "failed to resize node master-0: stopping VM: Azure capacity error",
		},
		{
			name: "resize VM fails",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
					a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
					a.EXPECT().VMResize(gomock.Any(), "master-0", desiredSize).
						Return(errors.New("Azure resize error")),
				)
			},
			wantErr: "failed to resize node master-0: resizing VM: Azure resize error",
		},
		{
			name: "start VM fails",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
					a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
					a.EXPECT().VMResize(gomock.Any(), "master-0", desiredSize).Return(nil),
					a.EXPECT().VMStartAndWait(gomock.Any(), "master-0").
						Return(errors.New("start failed")),
				)
			},
			wantErr: "failed to resize node master-0: starting VM: start failed",
		},
		{
			name: "uncordon fails",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
					k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
					a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
					a.EXPECT().VMResize(gomock.Any(), "master-0", desiredSize).Return(nil),
					a.EXPECT().VMStartAndWait(gomock.Any(), "master-0").Return(nil),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().CordonNode(gomock.Any(), "master-0", false).
						Return(errors.New("uncordon failure")),
				)
			},
			wantErr: "failed to resize node master-0: uncordoning node: uncordon failure",
		},
		{
			name: "pre-loop gate fails when node is not ready",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", desiredSize, running)), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", false), nil)
			},
			wantErr: "409: RequestNotAllowed: : Control plane node master-0 is not Ready. Resolve node health before resizing another master.",
		},
		{
			name: "pre-loop gate fails when node is unschedulable",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
					masterMachineListJSON(masterMachine("master-0", desiredSize, running)), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSONWithSchedulability("master-0", true, true), nil)
			},
			wantErr: "409: RequestNotAllowed: : Control plane node master-0 is unschedulable. Uncordon and verify the node before resizing another master.",
		},
		{
			name: "no control plane machines found",
			mocks: func(k *mock_adminactions.MockKubeActions, a *mock_adminactions.MockAzureActions) {
				k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(masterMachineListJSON(), nil)
			},
			wantErr: "409: RequestNotAllowed: : No control plane machines found. Resize cannot proceed.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k := mock_adminactions.NewMockKubeActions(ctrl)
			a := mock_adminactions.NewMockAzureActions(ctrl)
			tt.mocks(k, a)

			err := resizeControlPlane(ctx, log, k, a, desiredSize, true)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestUpdateMachineVMSize(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "success",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-0").
					Return(machineJSON("master-0", "Standard_D8s_v3"), nil)
				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, obj *unstructured.Unstructured) error {
						ts, found, err := unstructured.NestedString(obj.Object, "spec", "providerSpec", "value", "metadata", "creationTimestamp")
						if err != nil {
							t.Fatalf("unexpected error reading providerSpec metadata.creationTimestamp: %v", err)
						}
						if !found {
							t.Fatal("providerSpec metadata.creationTimestamp was not set")
						}
						if ts != "2024-01-01T00:00:00Z" {
							t.Fatalf("providerSpec metadata.creationTimestamp = %q, want %q", ts, "2024-01-01T00:00:00Z")
						}
						return nil
					})
			},
		},
		{
			name: "retries on failure",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-0").
						Return(machineJSON("master-0", "Standard_D8s_v3"), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(errors.New("conflict")),
					k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-0").
						Return(machineJSON("master-0", "Standard_D8s_v3"), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
				)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(k)

			err := updateMachineVMSize(ctx, k, "master-0", "Standard_D16s_v5")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestUpdateNodeInstanceTypeLabels(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_adminactions.MockKubeActions)
		wantErr string
	}{
		{
			name: "success",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", true), nil)
				k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "retries on failure",
			mocks: func(k *mock_adminactions.MockKubeActions) {
				gomock.InOrder(
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(errors.New("conflict")),
					k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
						Return(nodeJSON("master-0", true), nil),
					k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
				)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k := mock_adminactions.NewMockKubeActions(ctrl)
			tt.mocks(k)

			err := updateNodeInstanceTypeLabels(ctx, k, "master-0", "Standard_D16s_v5")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestAdminResizeControlPlane(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		vmSize         string
		deallocateVM   string
		fixture        func(f *testdatabase.Fixture)
		kubeMocks      func(*mock_adminactions.MockKubeActions)
		azureMocks     func(*mock_adminactions.MockAzureActions)
		wantStatusCode int
		wantError      string
	}

	addClusterDoc := func(f *testdatabase.Fixture) {
		f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
			Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						VMSize: api.VMSizeStandardD8sV3,
					},
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
					},
				},
			},
		})
	}

	addSubscriptionDoc := func(f *testdatabase.Fixture) {
		f.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubID,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: mockTenantID,
				},
			},
		})
	}

	for _, tt := range []*test{
		{
			name:         "happy path - prevalidation and no-op resize",
			vmSize:       "Standard_D8s_v3",
			deallocateVM: "true",
			resourceID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
				addSubscriptionDoc(f)
			},
			kubeMocks: func(k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "kube-apiserver").
					Return(healthyKubeAPIServerJSON(), nil).
					AnyTimes()
				k.EXPECT().
					KubeList(gomock.Any(), "Pod", "openshift-kube-apiserver").
					Return(healthyKubeAPIServerPodsJSON(), nil).
					AnyTimes()
				k.EXPECT().
					KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
					Return(healthyEtcdJSON(), nil).
					AnyTimes()
				k.EXPECT().
					KubeGet(gomock.Any(), "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName).
					Return(validServicePrincipalJSON(), nil).
					AnyTimes()
				k.EXPECT().
					KubeGet(gomock.Any(), "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster").
					Return(nil, kerrors.NewNotFound(schema.GroupResource{Group: "machine.openshift.io", Resource: "controlplanemachinesets"}, "cluster")).
					AnyTimes()

				running := "Running"
				k.EXPECT().
					KubeList(gomock.Any(), "Machine", machineNamespace).
					Return(masterMachineListJSON(
						masterMachine("master-0", "Standard_D8s_v3", running),
						masterMachine("master-1", "Standard_D8s_v3", running),
						masterMachine("master-2", "Standard_D8s_v3", running),
					), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").
					Return(nodeJSON("master-2", true), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").
					Return(nodeJSON("master-1", true), nil)
				k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
					Return(nodeJSON("master-0", true), nil)
			},
			azureMocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					VMGetSKUs(gomock.Any(), []string{"Standard_D8s_v3"}).
					Return(map[string]*armcompute.ResourceSKU{
						"Standard_D8s_v3": {
							Name:         pointerutils.ToPtr("Standard_D8s_v3"),
							ResourceType: pointerutils.ToPtr("virtualMachines"),
							Locations:    pointerutils.ToSlicePtr([]string{"eastus"}),
							LocationInfo: []*armcompute.ResourceSKULocationInfo{
								{
									Location: pointerutils.ToPtr("eastus"),
								},
							},
							Restrictions: pointerutils.ToSlicePtr([]armcompute.ResourceSKURestrictions{}),
							Capabilities: []*armcompute.ResourceSKUCapabilities{},
						},
					}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:         "invalid vm size",
			vmSize:       "Standard_Invalid_Size",
			deallocateVM: "true",
			resourceID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
				addSubscriptionDoc(f)
			},
			kubeMocks:      func(k *mock_adminactions.MockKubeActions) {},
			azureMocks:     func(a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: : The provided vmSize 'Standard_Invalid_Size' is unsupported for master.`,
		},
		{
			name:         "cluster not found",
			vmSize:       "Standard_D8s_v3",
			deallocateVM: "true",
			resourceID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addSubscriptionDoc(f)
			},
			kubeMocks:      func(k *mock_adminactions.MockKubeActions) {},
			azureMocks:     func(a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:         "subscription not found",
			vmSize:       "Standard_D8s_v3",
			deallocateVM: "true",
			resourceID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				addClusterDoc(f)
			},
			kubeMocks:      func(k *mock_adminactions.MockKubeActions) {},
			azureMocks:     func(a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf(`400: InvalidSubscriptionState: : Request is not allowed in unregistered subscription '%s'.`, mockSubID),
		},
		{
			name:           "invalid deallocateVM",
			vmSize:         "Standard_D8s_v3",
			deallocateVM:   "foo",
			resourceID:     testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture:        func(f *testdatabase.Fixture) {},
			kubeMocks:      func(k *mock_adminactions.MockKubeActions) {},
			azureMocks:     func(a *mock_adminactions.MockAzureActions) {},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidParameter: deallocateVM: The provided deallocateVM value 'foo' is invalid. Allowed values are 'true' or 'false'.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.kubeMocks(k)
			tt.azureMocks(a)

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx,
				ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup,
				api.APIs, &noop.Noop{}, &noop.Noop{},
				nil, nil, nil,
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
					return k, nil
				},
				func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
					return a, nil
				},
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Avoid creating real Azure quota clients in handler tests.
			f.validateResizeQuota = quotaCheckDisabled

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/resizecontrolplane?vmSize=%s&deallocateVM=%s", tt.resourceID, tt.vmSize, tt.deallocateVM),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
