package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestDeleteNic(t *testing.T) {
	ctx := context.Background()
	subscription := "00000000-0000-0000-0000-000000000000"
	clusterRG := "cluster-rg"
	nicName := "nic-name"
	location := "eastus"

	tests := []struct {
		name              string
		mocks             func(*mock_network.MockInterfacesClient)
		provisioningState *string
		wantErr           string
	}{
		{
			name: "nic is in succeeded provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
			provisioningState: to.StringPtr("SUCCEEDED"),
		},
		{
			name: "nic is in failed provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, gomock.Any()).Return(nil)
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
			provisioningState: to.StringPtr("FAILED"),
		},
		{
			name: "provisioning state is failed and CreateOrUpdateAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, gomock.Any()).Return(fmt.Errorf("Failed to update"))
			},
			provisioningState: to.StringPtr("FAILED"),
			wantErr:           "Failed to update",
		},
		{
			name: "DeleteAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(fmt.Errorf("Failed to delete"))
			},
			provisioningState: to.StringPtr("SUCCEEDED"),
			wantErr:           "Failed to delete",
		},
		{
			name: "provisioningState is nil",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
			provisioningState: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return(location)

			networkInterfaces := mock_network.NewMockInterfacesClient(controller)

			tt.mocks(networkInterfaces)

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, clusterRG),
							},
						},
					},
				},
				interfaces: networkInterfaces,
			}

			resource := mgmtfeatures.GenericResourceExpanded{
				Name:              to.StringPtr(nicName),
				ID:                to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscription, clusterRG, nicName)),
				ProvisioningState: tt.provisioningState,
			}

			err := m.deleteNic(ctx, resource)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error: '%s'", err.Error())
			}
		})
	}
}
