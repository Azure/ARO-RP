package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestValidateQuota(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name    string
		mocks   func(*test, *mock_compute.MockUsageClient, *mock_network.MockUsageClient)
		wantErr string
	}
	for _, tt := range []*test{
		{
			name: "allow when there's enough resources - limits set to exact requirements, offset by 100 of current value",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
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
				nuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(4),
							Limit:        to.Int64Ptr(10),
						},
					}, nil)
			},
		},
		{
			name:    "not enough cores",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of cores exceeded. Maximum allowed: 204, Current in use: 101, Additional requested: 104.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
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
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
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
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
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
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
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
		{
			name:    "not enough public ip addresses",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of PublicIPAddresses exceeded. Maximum allowed: 6, Current in use: 4, Additional requested: 3.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_network.MockUsageClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{}, nil)
				nuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtnetwork.Usage{
						{
							Name: &mgmtnetwork.UsageName{
								Value: to.StringPtr("PublicIPAddresses"),
							},
							CurrentValue: to.Int64Ptr(4),
							Limit:        to.Int64Ptr(6),
						},
					}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			computeUsageClient := mock_compute.NewMockUsageClient(controller)
			networkUsageClient := mock_network.NewMockUsageClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, computeUsageClient, networkUsageClient)
			}

			oc := &api.OpenShiftCluster{
				Location: "ocLocation",
				Properties: api.OpenShiftClusterProperties{
					Install: &api.Install{
						Phase: api.InstallPhaseBootstrap,
					},
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

			err := validateQuota(ctx, oc, networkUsageClient, computeUsageClient)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
