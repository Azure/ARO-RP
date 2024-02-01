package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, nic).Return(nil)
				networkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName).Return(nil)
			},
		},
		{
			name: "provisioning state is failed and CreateOrUpdateAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Failed
				networkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, nic).Return(fmt.Errorf("Failed to update"))
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestShouldDeleteResourceGroup(t *testing.T) {
	ctx := context.Background()
	subscription := "00000000-0000-0000-0000-000000000000"
	clusterName := "cluster"
	clusterRGName := "aro-cluster"
	clusterResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscription, clusterRGName, clusterName)
	managedRGName := "aro-managed-rg"

	errNotFound := autorest.DetailedError{
		StatusCode: http.StatusNotFound,
		Original: &azure.ServiceError{
			Code: "ResourceGroupNotFound",
		},
	}

	tests := []struct {
		name             string
		getResourceGroup mgmtfeatures.ResourceGroup
		getErr           error
		wantShouldDelete bool
		wantErr          string
	}{
		{
			name:             "get resource group - not found",
			getErr:           errNotFound,
			wantShouldDelete: false,
		},
		{
			name:             "get resource group - other error",
			getErr:           errors.New("generic err"),
			wantShouldDelete: false,
			wantErr:          "generic err",
		},
		{
			name:             "resource group not managed (nil)",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: nil},
			wantShouldDelete: false,
		},
		{
			name:             "resource group not managed (empty string)",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: to.StringPtr("")},
			wantShouldDelete: false,
		},
		{
			name:             "resource group not managed by cluster",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: to.StringPtr("/somethingelse")},
			wantShouldDelete: false,
		},
		{
			name:             "resource group managed by cluster",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: to.StringPtr(clusterResourceId)},
			wantShouldDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			resourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			resourceGroups.EXPECT().Get(gomock.Any(), gomock.Eq(managedRGName)).Return(tt.getResourceGroup, tt.getErr)

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceId,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, managedRGName),
							},
						},
					},
				},
				resourceGroups: resourceGroups,
			}

			shouldDelete, err := m.shouldDeleteResourceGroup(ctx, managedRGName)

			if shouldDelete != tt.wantShouldDelete {
				t.Errorf("wanted shouldDelete: %v but got %v", tt.wantShouldDelete, shouldDelete)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeleteResourceGroup(t *testing.T) {
	ctx := context.Background()
	subscription := "00000000-0000-0000-0000-000000000000"
	clusterName := "cluster"
	clusterRGName := "aro-cluster"
	clusterResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscription, clusterRGName, clusterName)
	managedRGName := "aro-managed-rg"

	errNotFound := autorest.DetailedError{
		StatusCode: http.StatusNotFound,
		Original: &azure.ServiceError{
			Code: "ResourceGroupNotFound",
		},
	}

	tests := []struct {
		name      string
		deleteErr error
		wantErr   string
	}{
		{
			name:      "not found",
			deleteErr: errNotFound,
		},
		{
			name:      "other error",
			deleteErr: errors.New("generic err"),
			wantErr:   "generic err",
		},
		{
			name: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			resourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			resourceGroups.EXPECT().DeleteAndWait(gomock.Any(), gomock.Eq(managedRGName)).Times(1).Return(tt.deleteErr)

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: clusterResourceId,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, managedRGName),
							},
						},
					},
				},
				resourceGroups: resourceGroups,
			}

			err := m.deleteResourceGroup(ctx, managedRGName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
