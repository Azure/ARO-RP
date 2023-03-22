package vmsscleaner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestRemoveFailedScaleset(t *testing.T) {
	ctx := context.Background()
	rg := "testRG"
	vmssToDelete := "newVMSS"
	servingVMSS := "oldVMSS"
	for _, tt := range []struct {
		name  string
		mocks func(*mock_compute.MockVirtualMachineScaleSetsClient)
		want  bool
	}{
		{
			name: "List failed, don't delete, don't retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{},
					errors.New("Something went wrong :("),
				)
			},
		},
		{
			name: "0 scalesets found, don't delete, retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{},
					nil,
				)
			},
			want: true,
		},
		{
			name: "1 scaleset found, different name from that in new deployment, don't delete, retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{
						{Name: to.StringPtr(servingVMSS)},
					},
					nil,
				)
			},
			want: true,
		},
		{
			name: "1 scaleset found, same name from that in new deployment, don't delete, don't retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{
						{Name: to.StringPtr(vmssToDelete)},
					},
					nil,
				)
			},
		},
		{
			name: "Target scaleset not found, don't delete, retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{
						{Name: to.StringPtr(servingVMSS)},
						{Name: to.StringPtr("otherVMSS")},
					},
					nil,
				)
			},
			want: true,
		},
		{
			name: "Target scaleset found, attempt deletion, deletion failed, don't retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{
						{Name: to.StringPtr(servingVMSS)},
						{Name: to.StringPtr(vmssToDelete)},
					},
					nil,
				)
				vmss.EXPECT().DeleteAndWait(ctx, rg, vmssToDelete).Return(errors.New("fake error"))
			},
		},
		{
			name: "Target scaleset found, attempt deletion, deletion succeeded, retry",
			mocks: func(vmss *mock_compute.MockVirtualMachineScaleSetsClient) {
				vmss.EXPECT().List(ctx, rg).Return(
					[]mgmtcompute.VirtualMachineScaleSet{
						{Name: to.StringPtr(servingVMSS)},
						{Name: to.StringPtr(vmssToDelete)},
					},
					nil,
				)
				vmss.EXPECT().DeleteAndWait(ctx, rg, vmssToDelete).Return(nil)
			},
			want: true,
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

			retry := c.RemoveFailedNewScaleset(ctx, rg, vmssToDelete)
			if retry != tt.want {
				t.Error(retry)
			}
		})
	}
}
