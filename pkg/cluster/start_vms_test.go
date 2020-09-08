package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
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
					{Name: to.StringPtr("starting-vm")},
					{Name: to.StringPtr("running-vm")},
					{Name: to.StringPtr("stopping-vm")},
					{Name: to.StringPtr("deallocating-vm")},
					{Name: to.StringPtr("stopped-vm")},
					{Name: to.StringPtr("deallocated-vm")},
				}
				getResults := []mgmtcompute.VirtualMachine{
					{
						Name: to.StringPtr("starting-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/starting")},
								},
							},
						},
					},
					{
						Name: to.StringPtr("running-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/running")},
								},
							},
						},
					},
					{
						Name: to.StringPtr("stopping-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/stopping")},
								},
							},
						},
					},
					{
						Name: to.StringPtr("deallocating-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/deallocating")},
								},
							},
						},
					},
					{
						Name: to.StringPtr("stopped-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/stopped")},
								},
							},
						},
					},
					{
						Name: to.StringPtr("deallocated-vm"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/deallocated")},
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
			name: "failed to start VMs",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: to.StringPtr("vm1")},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				vmClient.EXPECT().
					Get(gomock.Any(), clusterRGName, *vms[0].Name, mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{
						Name: to.StringPtr("vm1"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{
									{Code: to.StringPtr("PowerState/deallocated")},
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

			i := &manager{
				virtualmachines: vmClient,
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

			err := i.startVMs(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
