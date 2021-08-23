package vmsscleaner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestRemoveFailedScaleset(t *testing.T) {
	ctx := context.Background()
	rgName := "testRG"
	vmssName := "testVMSS"
	for _, tt := range []struct {
		name  string
		mocks func(*mock_compute.MockVirtualMachineScaleSetsClient)
		want  bool
	}{
		{
			name: "VMSS not found",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().Get(ctx, rgName, vmssName).Return(
					mgmtcompute.VirtualMachineScaleSet{},
					autorest.DetailedError{
						StatusCode: http.StatusNotFound,
					},
				)

			},
			want: true,
		},
		{
			name: "Found but not failed",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().Get(ctx, rgName, vmssName).Return(
					mgmtcompute.VirtualMachineScaleSet{
						VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
							ProvisioningState: to.StringPtr(string(mgmtcompute.ProvisioningStateSucceeded)),
						},
					},
					nil,
				)
			},
			want: false,
		},
		{
			name: "Found, failed, deletion failed",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().Get(ctx, rgName, vmssName).Return(
					mgmtcompute.VirtualMachineScaleSet{
						Name: to.StringPtr(vmssName),
						VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
							ProvisioningState: to.StringPtr(string(mgmtcompute.ProvisioningStateFailed)),
						},
					},
					nil,
				)
				vmss.EXPECT().DeleteAndWait(ctx, rgName, vmssName).Return(errors.New("fake error"))
			},
			want: false,
		},
		{
			name: "Found, failed, deletion succeeded",
			want: true,
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().Get(ctx, rgName, vmssName).Return(
					mgmtcompute.VirtualMachineScaleSet{
						Name: to.StringPtr(vmssName),
						VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
							ProvisioningState: to.StringPtr(string(mgmtcompute.ProvisioningStateFailed)),
						},
					},
					nil,
				)
				vmss.EXPECT().DeleteAndWait(ctx, rgName, vmssName).Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockVMSS := mock_compute.NewMockVirtualMachineScaleSetsClient(controller)
			tt.mocks(mockVMSS)

			c := cleaner{
				log:  logrus.NewEntry(logrus.StandardLogger()),
				vmss: mockVMSS,
			}

			deleted := c.RemoveFailedScaleset(ctx, rgName, vmssName)
			if deleted != tt.want {
				t.Error(deleted)
			}

		})
	}
}
