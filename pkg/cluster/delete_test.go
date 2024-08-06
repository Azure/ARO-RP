package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/msi-dataplane/pkg/store"
	mockkvclient "github.com/Azure/msi-dataplane/pkg/store/mock_kvclient"
	deprecratedgomock "github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
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
				networkInterfaces.EXPECT().Get(deprecratedgomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().DeleteAndWait(deprecratedgomock.Any(), clusterRG, nicName).Return(nil)
			},
		},
		{
			name: "nic is in failed provisioning state",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Failed
				networkInterfaces.EXPECT().Get(deprecratedgomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(deprecratedgomock.Any(), clusterRG, nicName, nic).Return(nil)
				networkInterfaces.EXPECT().DeleteAndWait(deprecratedgomock.Any(), clusterRG, nicName).Return(nil)
			},
		},
		{
			name: "provisioning state is failed and CreateOrUpdateAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Failed
				networkInterfaces.EXPECT().Get(deprecratedgomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().CreateOrUpdateAndWait(deprecratedgomock.Any(), clusterRG, nicName, nic).Return(fmt.Errorf("Failed to update"))
			},
			wantErr: "Failed to update",
		},
		{
			name: "nic no longer exists - do nothing",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				notFound := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}
				networkInterfaces.EXPECT().Get(deprecratedgomock.Any(), clusterRG, nicName, "").Return(nic, notFound)
			},
		},
		{
			name: "DeleteAndWait returns error",
			mocks: func(networkInterfaces *mock_network.MockInterfacesClient) {
				nic.InterfacePropertiesFormat.ProvisioningState = mgmtnetwork.Succeeded
				networkInterfaces.EXPECT().Get(deprecratedgomock.Any(), clusterRG, nicName, "").Return(nic, nil)
				networkInterfaces.EXPECT().DeleteAndWait(deprecratedgomock.Any(), clusterRG, nicName).Return(fmt.Errorf("Failed to delete"))
			},
			wantErr: "Failed to delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := deprecratedgomock.NewController(t)
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
			controller := deprecratedgomock.NewController(t)
			defer controller.Finish()

			resourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			resourceGroups.EXPECT().Get(deprecratedgomock.Any(), deprecratedgomock.Eq(managedRGName)).Return(tt.getResourceGroup, tt.getErr)

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
			controller := deprecratedgomock.NewController(t)
			defer controller.Finish()

			resourceGroups := mock_features.NewMockResourceGroupsClient(controller)
			resourceGroups.EXPECT().DeleteAndWait(deprecratedgomock.Any(), deprecratedgomock.Eq(managedRGName)).Times(1).Return(tt.deleteErr)

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

func TestDisconnectSecurityGroup(t *testing.T) {
	subscription := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "test-rg"
	nsgId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/test-nsg", subscription, resourceGroup)

	tests := []struct {
		name    string
		mocks   func(*mock_armnetwork.MockSecurityGroupsClient, *mock_subnet.MockManager)
		wantErr string
	}{
		{
			name: "empty security group",
			mocks: func(securityGroups *mock_armnetwork.MockSecurityGroupsClient, subnets *mock_subnet.MockManager) {
				securityGroup := armnetwork.SecurityGroupsClientGetResponse{
					SecurityGroup: armnetwork.SecurityGroup{
						ID: ptr.To(nsgId),
						Properties: &armnetwork.SecurityGroupPropertiesFormat{
							Subnets: []*armnetwork.Subnet{},
						},
					},
				}
				securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(securityGroup, nil)
				subnets.EXPECT().CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "disconnects subnets",
			mocks: func(securityGroups *mock_armnetwork.MockSecurityGroupsClient, subnets *mock_subnet.MockManager) {
				subnetId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscription, resourceGroup)
				securityGroup := armnetwork.SecurityGroupsClientGetResponse{
					SecurityGroup: armnetwork.SecurityGroup{
						ID: ptr.To(nsgId),
						Properties: &armnetwork.SecurityGroupPropertiesFormat{
							Subnets: []*armnetwork.Subnet{
								{
									ID: ptr.To(subnetId),
								},
							},
						},
					},
				}
				securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(securityGroup, nil)
				subnets.EXPECT().Get(gomock.Any(), subnetId).Times(1).Return(&mgmtnetwork.Subnet{
					ID: ptr.To(subnetId),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: ptr.To(nsgId),
						},
					},
				}, nil)
				subnets.EXPECT().CreateOrUpdate(gomock.Any(), subnetId, &mgmtnetwork.Subnet{
					ID:                     ptr.To(subnetId),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{},
				}).Times(1).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := deprecratedgomock.NewController(t)
			defer controller.Finish()

			securityGroups := mock_armnetwork.NewMockSecurityGroupsClient(controller)
			subnets := mock_subnet.NewMockManager(controller)

			tt.mocks(securityGroups, subnets)

			m := manager{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				armSecurityGroups: securityGroups,
				subnet:            subnets,
			}

			ctx := context.Background()
			err := m.disconnectSecurityGroup(ctx, nsgId)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestDeleteClusterMsiCertificate(t *testing.T) {
	ctx := context.Background()
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mockkvclient.MockKeyVaultClient)
		wantErr string
	}{
		{
			name: "success - cluster doc has nil Identity",
			doc: &api.OpenShiftClusterDocument{
				ID:               mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{},
			},
		},
		{
			name: "success - cluster doc has non-nil Identity but no Identity.UserAssignedIdentities",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{},
				},
			},
		},
		{
			name: "success - cluster doc has non-nil Identity but empty Identity.UserAssignedIdentities",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{},
					},
				},
			},
		},
		{
			name: "error - error getting cluster MSI secret name (this theoretically won't happen, but...)",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							"not a valid MI resource ID": api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'not a valid MI resource ID' must start with '/'",
		},
		{
			name: "error - error deleting cluster MSI certificate from key vault",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			mocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), fmt.Sprintf("%s-%s", mockGuid, miName), gomock.Any()).Times(1).Return(azsecrets.DeleteSecretResponse{}, fmt.Errorf("error in DeleteSecret"))
			},
			wantErr: "error in DeleteSecret",
		},
		{
			name: "success - successfully delete certificate",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			mocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), fmt.Sprintf("%s-%s", mockGuid, miName), gomock.Any()).Times(1).Return(azsecrets.DeleteSecretResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
			}

			mockKvClient := mockkvclient.NewMockKeyVaultClient(controller)
			if tt.mocks != nil {
				tt.mocks(mockKvClient)
			}

			m.clusterMsiKeyVaultStore = store.NewMsiKeyVaultStore(mockKvClient)

			err := m.deleteClusterMsiCertificate(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
