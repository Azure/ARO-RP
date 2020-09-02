package virtualmachines

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestListStopped(t *testing.T) {
	ctx := context.Background()
	clusterRGName := "test-cluster"

	for _, tt := range []struct {
		name    string
		mock    func(vmClient *mock_compute.MockVirtualMachinesClient)
		wantErr string
		want    []mgmtcompute.VirtualMachine
	}{
		{
			// Hopefully will never happen, but it's very easy to dereference a nil when digging out power statuses
			name: "vms do not have statuses",
			mock: func(vmClient *mock_compute.MockVirtualMachinesClient) {
				vms := []mgmtcompute.VirtualMachine{
					{Name: to.StringPtr("nil-status-code")},
					{Name: to.StringPtr("nil-statuses")},
					{Name: to.StringPtr("nil-instanceview")},
					{Name: to.StringPtr("nil-virtualmachineproperties")},
				}
				getResults := []mgmtcompute.VirtualMachine{
					{
						Name: to.StringPtr("nil-status-code"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{
								Statuses: &[]mgmtcompute.InstanceViewStatus{{}},
							},
						},
					},
					{
						Name: to.StringPtr("nil-statuses"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
							InstanceView: &mgmtcompute.VirtualMachineInstanceView{},
						},
					},
					{
						Name:                     to.StringPtr("nil-instanceview"),
						VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{},
					},
					{
						Name: to.StringPtr("nil-virtualmachineproperties"),
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
					{Name: to.StringPtr("vm1")},
				}

				vmClient.EXPECT().List(gomock.Any(), clusterRGName).Return(vms, nil)
				vmClient.EXPECT().
					Get(gomock.Any(), clusterRGName, *vms[0].Name, mgmtcompute.InstanceView).
					Return(mgmtcompute.VirtualMachine{}, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)

			tt.mock(vmClient)

			result, err := ListStopped(ctx, vmClient, clusterRGName)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.want != nil && !reflect.DeepEqual(result, tt.want) {
				t.Error(result)
			}
		})
	}
}
