package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestQuotaCheck(t *testing.T) {
	ctx := context.Background()

	oc := &api.OpenShiftCluster{
		Location: "ocLocation",
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{
				VMSize: "Standard_D8s_v3",
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					VMSize: "Standard_D8s_v3",
					Count:  10,
				},
			},
		},
	}

	dv := openShiftClusterDynamicValidator{}
	type test struct {
		name    string
		mocks   func(*test, *mock_compute.MockUsageClient)
		wantErr string
	}
	for _, tt := range []*test{
		{
			name: "allow when there's enough resources - limits set to exact requirements, offset by 100 of current value",
			mocks: func(tt *test, uc *mock_compute.MockUsageClient) {
				uc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("cores"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(204),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("virtualMachines"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(113),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("standardDSv3Family"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(204),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("PremiumDiskCount"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(113),
						},
					}, nil)
			},
		},
		{
			name:    "not enough cores",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of cores exceeded. Maximum allowed: 204, Current in use: 101, Additional requested: 104.",
			mocks: func(tt *test, uc *mock_compute.MockUsageClient) {
				uc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("cores"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(204),
						},
					}, nil)
			},
		},
		{
			name:    "not enough virtualMachines",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of virtualMachines exceeded. Maximum allowed: 113, Current in use: 101, Additional requested: 13.",
			mocks: func(tt *test, uc *mock_compute.MockUsageClient) {
				uc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("virtualMachines"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(113),
						},
					}, nil)
			},
		},
		{
			name:    "not enough standardDSv3Family",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of standardDSv3Family exceeded. Maximum allowed: 204, Current in use: 101, Additional requested: 104.",
			mocks: func(tt *test, uc *mock_compute.MockUsageClient) {
				uc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("standardDSv3Family"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(204),
						},
					}, nil)
			},
		},
		{
			name:    "not enough premium disks",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of PremiumDiskCount exceeded. Maximum allowed: 113, Current in use: 101, Additional requested: 13.",
			mocks: func(tt *test, uc *mock_compute.MockUsageClient) {
				uc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("PremiumDiskCount"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(113),
						},
					}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			usageClient := mock_compute.NewMockUsageClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, usageClient)
			}

			err := dv.validateQuotas(ctx, oc, usageClient)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
