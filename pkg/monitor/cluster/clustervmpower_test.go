package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func mockVM(name, id, powerState *string) mgmtcompute.VirtualMachine {
	return mgmtcompute.VirtualMachine{
		Name: name,
		ID:   id,
		VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
			InstanceView: &mgmtcompute.VirtualMachineInstanceView{
				Statuses: &[]mgmtcompute.InstanceViewStatus{
					{
						Code: powerState,
					},
				},
			},
		},
	}
}

func TestEmitVMPowerStatus(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockRGName := "test-cluster"
	mockVMName := to.StringPtr("mockVMName")
	mockVMID := to.StringPtr("mockVMID")
	powerStateDeallocated := to.StringPtr("PowerState/deallocated")
	powerStateStopped := to.StringPtr("PowerState/stopped")
	powerStateRunning := to.StringPtr("PowerState/running")

	ctx := context.Background()

	type test struct {
		name  string
		mocks func(*test, *mock_compute.MockVirtualMachinesClient, *mock_metrics.MockInterface)
		want  bool
	}

	for _, tt := range []*test{
		{
			name: "Has a deallocated VM",
			mocks: func(tt *test, vmClient *mock_compute.MockVirtualMachinesClient, m *mock_metrics.MockInterface) {
				deallocatedVM := mockVM(mockVMName, mockVMID, powerStateDeallocated)
				runningVM := mockVM(mockVMName, mockVMID, powerStateRunning)

				vmClient.EXPECT().List(ctx, mockRGName).
					Return([]mgmtcompute.VirtualMachine{deallocatedVM, runningVM}, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(deallocatedVM, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(runningVM, nil)

				m.EXPECT().EmitGauge("vmpower.conditions", int64(1), map[string]string{
					"id":     *mockVMID,
					"status": *powerStateDeallocated,
				})
			},
			want: true,
		},
		{
			name: "Has a stopped VM",
			mocks: func(tt *test, vmClient *mock_compute.MockVirtualMachinesClient, m *mock_metrics.MockInterface) {
				stoppedVM := mockVM(mockVMName, mockVMID, powerStateStopped)
				runningVM := mockVM(mockVMName, mockVMID, powerStateRunning)

				vmClient.EXPECT().List(ctx, mockRGName).
					Return([]mgmtcompute.VirtualMachine{stoppedVM, runningVM}, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(stoppedVM, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(runningVM, nil)

				m.EXPECT().EmitGauge("vmpower.conditions", int64(1), map[string]string{
					"id":     *mockVMID,
					"status": *powerStateStopped,
				})
			},
			want: true,
		},
		{
			name: "Only has running VMs",
			mocks: func(tt *test, vmClient *mock_compute.MockVirtualMachinesClient, m *mock_metrics.MockInterface) {
				runningVM1 := mockVM(mockVMName, mockVMID, powerStateRunning)
				runningVM2 := mockVM(mockVMName, mockVMID, powerStateRunning)

				vmClient.EXPECT().List(ctx, mockRGName).
					Return([]mgmtcompute.VirtualMachine{runningVM1, runningVM2}, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(runningVM1, nil)

				vmClient.EXPECT().Get(gomock.Any(), mockRGName, *mockVMName, mgmtcompute.InstanceView).
					Return(runningVM2, nil)
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			controller := gomock.NewController(t)
			defer controller.Finish()

			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)
			m := mock_metrics.NewMockInterface(controller)

			tt.mocks(tt, vmClient, m)

			mon := &Monitor{
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", mockSubID, mockRGName),
						},
					},
				},
				m:        m,
				vmClient: vmClient,
			}

			hasStoppedVMs, err := mon.emitStoppedVMPowerStatus(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if hasStoppedVMs != tt.want {
				t.Fatal(tt.want)
			}
		})
	}
}
