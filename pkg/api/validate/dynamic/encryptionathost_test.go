package dynamic

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

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateEncryptionAtHost(t *testing.T) {
	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(env *mock_env.MockInterface)
		wantErr string
	}{
		{
			name: "encryption at host disabled",
			oc:   &api.OpenShiftCluster{},
		},
		{
			name: "encryption at host enabled with valid VM SKU",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardD8sV3,
					},
					WorkerProfiles: []api.WorkerProfile{{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardD4asV4,
					}},
				},
			},
			mocks: func(env *mock_env.MockInterface) {
				env.EXPECT().VMSku(string(api.VMSizeStandardD8sV3)).
					Return(&mgmtcompute.ResourceSku{
						Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
							{Name: to.StringPtr("EncryptionAtHostSupported"), Value: to.StringPtr("True")},
						}),
					}, nil)
				env.EXPECT().VMSku(string(api.VMSizeStandardD4asV4)).
					Return(&mgmtcompute.ResourceSku{
						Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
							{Name: to.StringPtr("EncryptionAtHostSupported"), Value: to.StringPtr("True")},
						}),
					}, nil)
			},
		},
		{
			name: "encryption at host enabled with unsupported master VM SKU",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardM128ms,
					},
					WorkerProfiles: []api.WorkerProfile{{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardD4asV4,
					}},
				},
			},
			mocks: func(env *mock_env.MockInterface) {
				env.EXPECT().VMSku(string(api.VMSizeStandardM128ms)).
					Return(&mgmtcompute.ResourceSku{
						Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
							{Name: to.StringPtr("EncryptionAtHostSupported"), Value: to.StringPtr("False")},
						}),
					}, nil)
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.encryptionAtHost: VM SKU 'Standard_M128ms' does not support encryption at host.",
		},
		{
			name: "encryption at host enabled with unsupported worker VM SKU",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardD8sV3,
					},
					WorkerProfiles: []api.WorkerProfile{{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardM128ms,
					}},
				},
			},
			mocks: func(env *mock_env.MockInterface) {
				env.EXPECT().VMSku(string(api.VMSizeStandardD8sV3)).
					Return(&mgmtcompute.ResourceSku{
						Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
							{Name: to.StringPtr("EncryptionAtHostSupported"), Value: to.StringPtr("True")},
						}),
					}, nil)
				env.EXPECT().VMSku(string(api.VMSizeStandardM128ms)).
					Return(&mgmtcompute.ResourceSku{
						Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
							{Name: to.StringPtr("EncryptionAtHostSupported"), Value: to.StringPtr("False")},
						}),
					}, nil)
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].encryptionAtHost: VM SKU 'Standard_M128ms' does not support encryption at host.",
		},
		{
			name: "encryption at host enabled with unknown VM SKU",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           "invalid",
					},
					WorkerProfiles: []api.WorkerProfile{{
						EncryptionAtHost: api.EncryptionAtHostEnabled,
						VMSize:           api.VMSizeStandardM128ms,
					}},
				},
			},
			mocks: func(env *mock_env.MockInterface) {
				env.EXPECT().VMSku("invalid").
					Return(nil, errors.New("fake error"))
			},
			wantErr: "fake error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)

			if tt.mocks != nil {
				tt.mocks(_env)
			}

			dv := &dynamic{
				env:            _env,
				authorizerType: AuthorizerClusterServicePrincipal,
				log:            logrus.NewEntry(logrus.StandardLogger()),
			}

			err := dv.ValidateEncryptionAtHost(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
