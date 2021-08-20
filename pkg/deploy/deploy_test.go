package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestGetParameters(t *testing.T) {
	databaseAccountName := to.StringPtr("databaseAccountName")
	adminApiCaBundle := to.StringPtr("adminApiCaBundle")
	extraClusterKeyVaultAccessPolicies := []interface{}{"a", "b", 1}
	for _, tt := range []struct {
		name   string
		ps     map[string]interface{}
		config Configuration
		want   arm.Parameters
	}{
		{
			name: "when no parameters are present only default is returned",
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
		{
			name: "when all parameters present, everything is copied",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName:                databaseAccountName,
				AdminAPICABundle:                   adminApiCaBundle,
				ExtraClusterKeyvaultAccessPolicies: extraClusterKeyVaultAccessPolicies,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
					"extraClusterKeyvaultAccessPolicies": {
						Value: extraClusterKeyVaultAccessPolicies,
					},
					"adminApiCaBundle": {
						Value: adminApiCaBundle,
					},
				},
			},
		},
		{
			name: "when parameters with nil config are present, they are not returned",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName: databaseAccountName,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
				},
			},
		},
		{
			name: "when nil slice parameter is present it is skipped",
			ps: map[string]interface{}{
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
		{
			name: "when malformed parameter is present, it is skipped",
			ps: map[string]interface{}{
				"dutabaseAccountName": nil,
			},
			config: Configuration{
				DatabaseAccountName: databaseAccountName,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := deployer{
				config: &RPConfig{Configuration: &tt.config},
			}

			got := d.getParameters(tt.ps)

			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("%#v", got)
			}

		})
	}
}

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

			d := deployer{
				log:  logrus.NewEntry(logrus.StandardLogger()),
				vmss: mockVMSS,
			}

			deleted := d.removeFailedScaleset(ctx, rgName, vmssName)
			if deleted != tt.want {
				t.Error(deleted)
			}

		})
	}
}
