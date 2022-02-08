package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func getNic(subscription, resourceGroup, location, nicName string) mgmtnetwork.Interface {
	return mgmtnetwork.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(location),
		ID:       to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscription, resourceGroup, nicName)),
		InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
			ProvisioningState: mgmtnetwork.Failed,
		},
	}
}

func TestReconcileFailedNic(t *testing.T) {
	ctx := context.Background()
	subscription := "00000000-0000-0000-0000-000000000000"
	clusterRG := "cluster-rg"
	nicName := "nic-name"
	location := "eastus"

	tests := []struct {
		name    string
		mocks   func(*mock_network.MockInterfacesClient)
		wantErr string
	}{
		{
			name: "successfully reconcile nic",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic := getNic(subscription, clusterRG, location, nicName)
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, gomock.Any()).Return(nil)
			},
		},
		{
			name: "nic not in failed provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic := getNic(subscription, clusterRG, location, nicName)
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Succeeded
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
			},
			wantErr: fmt.Sprintf("skipping nic '%s' because it is not in a failed provisioning state", nicName),
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

			a := azureActions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, clusterRG),
						},
					},
				},
				networkInterfaces: networkInterfaces,
			}

			err := a.NICReconcileFailedState(ctx, nicName)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("got error: '%s'\nwant error: '%s'", err.Error(), tt.wantErr)
			}
		})
	}
}
