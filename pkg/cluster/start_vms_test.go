package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestStartVMs(t *testing.T) {
	ctx := context.Background()
	clusterRGName := "test-cluster"

	for _, tt := range []struct {
		name    string
		mock    func(vmClient *mock_compute.MockVirtualMachinesClient)
		wantErr string
	}{
		{
			name: "start only stopped and deallocated VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: pointerutils.ToPtr("starting-vm")},
					{Name: pointerutils.ToPtr("running-vm")},
					{Name: pointerutils.ToPtr("stopping-vm")},
					{Name: pointerutils.ToPtr("deallocating-vm")},
					{Name: pointerutils.ToPtr("stopped-vm")},
					{Name: pointerutils.ToPtr("deallocated-vm")},
				}
				getResults := []mgmtcompute.VirtualMachine{
					{
						Name: pointerutils.ToPtr("starting-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/starting")},
								},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("running-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/running")},
								},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("stopping-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/stopping")},
								},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("deallocating-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/deallocating")},
								},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("stopped-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/stopped")},
								},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("deallocated-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/deallocated")},
								},
							},
						},
					},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				for idx, vm := range vms {
					vmClient.EXPECT().
						Get(gomock.Any(), clusterRGName, *vm.Name, mgmtcompute.InstanceView).
						Return(getResults[idx], nil)
				}

				vmClient.EXPECT().StartAndWait(gomock.Any(), clusterRGName, "stopped-vm").Return(nil)
				vmClient.EXPECT().StartAndWait(gomock.Any(), clusterRGName, "deallocated-vm").Return(nil)
			},
		},
		{
			// Hopefully will never happen, but it's very easy to dereference a nil when digging out power statuses
			name: "vms do not have statuses",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: pointerutils.ToPtr("nil-status-code")},
					{Name: pointerutils.ToPtr("nil-statuses")},
					{Name: pointerutils.ToPtr("nil-instanceview")},
					{Name: pointerutils.ToPtr("nil-virtualmachineproperties")},
				}
				getResults := []mgmtcompute.VirtualMachine{
					{
						Name: pointerutils.ToPtr("nil-status-code"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{{}},
							},
						},
					},
					{
						Name: pointerutils.ToPtr("nil-statuses"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{},
						},
					},
					{
						Name:                     pointerutils.ToPtr("nil-instanceview"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
					{
						Name: pointerutils.ToPtr("nil-virtualmachineproperties"),
					},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				for idx, vm := range vms {
					vmClient.EXPECT().
						Get(gomock.Any(), clusterRGName, *vm.Name, mgmtcompute.InstanceView).
						Return(getResults[idx], nil)
				}
			},
		},
		{
			name: "failed to list VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "failed to get VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: pointerutils.ToPtr("vm1")},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				vmClient.EXPECT().
					Get(gomock.Any(), clusterRGName, *vms[0].Name, mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{}, errors.New("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "failed to start VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: pointerutils.ToPtr("vm1")},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				vmClient.EXPECT().
					Get(gomock.Any(), clusterRGName, *vms[0].Name, mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{
						Name: pointerutils.ToPtr("vm1"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: pointerutils.ToPtr("PowerState/deallocated")},
								},
							},
						},
					}, nil)
				vmClient.EXPECT().StartAndWait(gomock.Any(), clusterRGName, *vms[0].Name).Return(errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)

			tt.mock(vmClient)

			m := &manager{
				virtualMachines: vmClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
							},
						},
					},
				},
			}

			err := m.startVMs(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
