package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestCheckCustomDNS(t *testing.T) {
	ctx := context.Background()
	subscriptionID := "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"

	tests := []struct {
		name    string
		mocks   func(*mock_network.MockVirtualNetworksClient)
		wantErr string
	}{
		{
			name: "default dns",
			mocks: func(vnetc *mock_network.MockVirtualNetworksClient) {
				vnetc.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(
					mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{},
							},
						},
					}, nil)
			},
		},
		{
			name: "custom dns",
			mocks: func(vnetc *mock_network.MockVirtualNetworksClient) {
				vnetc.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(
					mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{"1.1.1.1"},
							},
						},
					}, nil)
			},
			wantErr: "not upgrading: custom DNS is set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
			if tt.mocks != nil {
				tt.mocks(vnetClient)
			}

			i := &manager{
				vnet: vnetClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
							},
						},
					},
				},
			}

			err := checkCustomDNS(ctx, i.doc, i.vnet)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
