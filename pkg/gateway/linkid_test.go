package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGatewayVerification(t *testing.T) {
	for _, tt := range []struct {
		name          string
		host          string
		idParam       string
		wantId        string
		wantIsAllowed bool
		wantErr       string
		deleting      bool
		allowList     map[string]struct{}
	}{
		{
			name:          "accepted id=1",
			host:          "account1.blob.storageEndpointSuffix",
			idParam:       "1",
			wantId:        "1",
			wantIsAllowed: true,
		},
		{
			name:          "accepted id=2",
			host:          "account2.blob.storageEndpointSuffix",
			idParam:       "2",
			wantId:        "2",
			wantIsAllowed: true,
		},
		{
			name:          "accepted allowlist",
			host:          "redhat.com",
			idParam:       "2",
			wantId:        "2",
			wantIsAllowed: true,
			allowList:     map[string]struct{}{"redhat.com": {}},
		},
		{
			name:          "middle part not valid",
			host:          "account1.notblob.storageEndpointSuffix",
			idParam:       "1",
			wantId:        "1",
			wantIsAllowed: false,
		},
		{
			name:          "suffix not valid",
			host:          "account1.blob.notstorageEndpointSuffix",
			idParam:       "1",
			wantId:        "1",
			wantIsAllowed: false,
		},
		{
			name:          "no gateway",
			host:          "account1.blob.notstorageEndpointSuffix",
			idParam:       "notinthemap",
			wantErr:       "gateway record not found for linkID notinthemap",
			wantId:        "",
			wantIsAllowed: false,
		},
		{
			name:          "no host",
			idParam:       "1",
			wantId:        "1",
			wantIsAllowed: false,
		},
		{
			name:     "gateway deleting",
			host:     "account2.blob.storageEndpointSuffix",
			wantErr:  "gateway for linkId deleting is being deleted",
			idParam:  "deleting",
			wantId:   "deleting",
			deleting: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockController := gomock.NewController(t)
			defer mockController.Finish()

			gatewayMap := map[string]*api.Gateway{
				"1":        {ID: "1", StorageSuffix: "suffix-1", ImageRegistryStorageAccountName: "account1"},
				"2":        {ID: "2", StorageSuffix: "suffix-2", ImageRegistryStorageAccountName: "account2"},
				"deleting": {ID: "deleting", StorageSuffix: "suffix-5", ImageRegistryStorageAccountName: "account5", Deleting: true},
			}

			mockCore := mock_env.NewMockCore(mockController)
			mockCore.
				EXPECT().
				Environment().
				Return(&azureclient.AROEnvironment{Environment: azure.Environment{StorageEndpointSuffix: "storageEndpointSuffix"}}).
				AnyTimes()

			mock_metrics := mock_metrics.NewMockEmitter(mockController)

			if tt.host == "" {
				mock_metrics.EXPECT().EmitGauge("gateway.nohost", int64(1), map[string]string{
					"linkid": tt.idParam,
					"action": "denied",
				}).MinTimes(1)
			}

			gateway := gateway{
				m:         mock_metrics,
				gateways:  gatewayMap,
				env:       mockCore,
				allowList: tt.allowList,
			}

			gatewayID, isAllowed, err := gateway.gatewayVerification(tt.host, tt.idParam)

			if gatewayID != tt.wantId {
				t.Error(gatewayID)
			}

			if isAllowed != tt.wantIsAllowed {
				t.Error(isAllowed)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
