package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func getNic(subscription, resourceGroup, location, nicName string) armnetwork.Interface {
	return armnetwork.Interface{
		Name:     pointerutils.ToPtr(nicName),
		Location: pointerutils.ToPtr(location),
		ID:       pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscription, resourceGroup, nicName)),
		Properties: &armnetwork.InterfacePropertiesFormat{
			ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateFailed),
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
		mocks   func(*mock_armnetwork.MockInterfacesClient)
		wantErr string
	}{
		{
			name: "successfully reconcile nic",
			mocks: func(networkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic := getNic(subscription, clusterRG, location, nicName)
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{Interface: nic}, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, nic, nil).Return(nil)
			},
		},
		{
			name: "nic not in failed provisioning state",
			mocks: func(networkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic := getNic(subscription, clusterRG, location, nicName)
				nic.Properties.ProvisioningState = pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded)
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(armnetwork.InterfacesClientGetResponse{Interface: nic}, nil)
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

			networkInterfaces := mock_armnetwork.NewMockInterfacesClient(controller)

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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
