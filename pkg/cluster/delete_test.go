package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
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
	resourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscription, clusterRG, nicName)

	nic := mgmtnetwork.Interface{
		Name:                      &nicName,
		Location:                  &location,
		ID:                        &resourceId,
		InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{},
	}

	tests := []struct {
		name    string
		mocks   func(*mock_network.MockInterfacesClient)
		wantErr string
	}{
		{
			name: "nic is in succeeded provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Succeeded
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
		},
		{
			name: "nic is in failed provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Failed
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, gomock.Any()).Return(nil)
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
		},
		{
			name: "provisioning state is failed and CreateOrUpdateAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Failed
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, gomock.Any()).Return(fmt.Errorf("Failed to update"))
			},
			wantErr: "Failed to update",
		},
		{
			name: "nic no longer exists - do nothing",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				notFound := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, notFound)
			},
		},
		{
			name: "DeleteAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Succeeded
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(fmt.Errorf("Failed to delete"))
			},
			wantErr: "Failed to delete",
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

			err := m.deleteNic(ctx, nicName)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error: '%s'", err.Error())
			}
		})
	}
}
