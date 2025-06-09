package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateQuota(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name    string
		mocks   func(*test, *mock_compute.MockUsageClient, *mock_armnetwork.MockUsagesClient)
		wantErr string
	}
	for _, tt := range []*test{
		{
			name: "allow when there's enough resources - limits set to exact requirements, offset by 100 of current value",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("cores"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(212),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("virtualMachines"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(114),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("standardDSv3Family"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(212),
						},
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("PremiumDiskCount"),
							},
							CurrentValue: to.Int32Ptr(100),
							Limit:        to.Int64Ptr(114),
						},
					}, nil)
				nuc.EXPECT().
					List(ctx, "ocLocation", nil).
					Return([]*sdknetwork.Usage{
						{
							Name: &sdknetwork.UsageName{
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
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of cores exceeded. Maximum allowed: 212, Current in use: 101, Additional requested: 112.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("cores"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(212),
						},
					}, nil)
			},
		},
		{
			name:    "not enough virtualMachines",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of virtualMachines exceeded. Maximum allowed: 114, Current in use: 101, Additional requested: 14.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("virtualMachines"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(114),
						},
					}, nil)
			},
		},
		{
			name:    "not enough standardDSv3Family",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of standardDSv3Family exceeded. Maximum allowed: 212, Current in use: 101, Additional requested: 112.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("standardDSv3Family"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(212),
						},
					}, nil)
			},
		},
		{
			name:    "not enough premium disks",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of PremiumDiskCount exceeded. Maximum allowed: 114, Current in use: 101, Additional requested: 14.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{
						{
							Name: &mgmtcompute.UsageName{
								Value: to.StringPtr("PremiumDiskCount"),
							},
							CurrentValue: to.Int32Ptr(101),
							Limit:        to.Int64Ptr(114),
						},
					}, nil)
			},
		},
		{
			name:    "not enough public ip addresses",
			wantErr: "400: ResourceQuotaExceeded: : Resource quota of PublicIPAddresses exceeded. Maximum allowed: 6, Current in use: 4, Additional requested: 3.",
			mocks: func(tt *test, cuc *mock_compute.MockUsageClient, nuc *mock_armnetwork.MockUsagesClient) {
				cuc.EXPECT().
					List(ctx, "ocLocation").
					Return([]mgmtcompute.Usage{}, nil)
				nuc.EXPECT().
					List(ctx, "ocLocation", nil).
					Return([]*sdknetwork.Usage{
						{
							Name: &sdknetwork.UsageName{
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
			networkUsageClient := mock_armnetwork.NewMockUsagesClient(controller)
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
