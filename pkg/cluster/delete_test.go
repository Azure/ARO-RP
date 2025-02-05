package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
			controller := gomock.NewController(t)
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
	secretName := dataplane.ManagedIdentityCredentialsStoragePrefix + mockGuid
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(mockManager *mock_keyvault.MockManager)
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
					Identity: &api.ManagedServiceIdentity{},
				},
			},
		},
		{
			name: "success - cluster doc has non-nil Identity but empty Identity.UserAssignedIdentities",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{},
					},
				},
			},
		},
		{
			name: "error - error deleting cluster MSI certificate from key vault",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			mocks: func(kvclient *mock_keyvault.MockManager) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), secretName).Times(1).Return(keyvault.DeletedSecretBundle{}, fmt.Errorf("error in DeleteSecret"))
			},
			wantErr: "error in DeleteSecret",
		},
		{
			name: "success - successfully delete certificate",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			mocks: func(kvclient *mock_keyvault.MockManager) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), secretName).Times(1).Return(keyvault.DeletedSecretBundle{}, nil)
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

			mockKvClient := mock_keyvault.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(mockKvClient)
			}

			m.clusterMsiKeyVaultStore = mockKvClient

			err := m.deleteClusterMsiCertificate(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDeleteFederatedCredentials(t *testing.T) {
	ctx := context.Background()
	docID := "00000000-0000-0000-0000-000000000000"
	clusterID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/fakeResourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/fakeCluster"
	clusterResourceId, _ := azure.ParseResourceID(clusterID)
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	identityIDPrefix := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/", mockGuid, clusterRGName)

	oidcIssuer := "https://fakeissuer.fakedomain/fakecluster"

	ccmServiceAccountName := "system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"
	ccmIdentityResourceId, _ := azure.ParseResourceID(fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"))
	ingressServiceAccountName := "system:serviceaccount:openshift-ingress-operator:ingress-operator"
	ingressIdentityResourceId, _ := azure.ParseResourceID(fmt.Sprintf("%s/%s", identityIDPrefix, "cio"))

	tests := []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mock_armmsi.MockFederatedIdentityCredentialsClient)
		wantErr string
	}{
		{
			name: "success - cluster doc has nil PlatformWorkloadIdentities",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
						},
					},
				},
			},
		},
		{
			name: "success - cluster doc has non-nil but empty PlatformWorkloadIdentities",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo:              ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
						},
					},
				},
			},
		},
		{
			name: "success - cluster doc has no oidc issuer so no actions performed",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: nil,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
								"ClusterIngressOperator": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "cio"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "success - identities have no federated credentials",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
								"ClusterIngressOperator": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "cio"),
								},
							},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
		},
		{
			name: "success - successfully delete federated credentials",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
								"ClusterIngressOperator": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "cio"),
								},
							},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				ccmFedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, ccmIdentityResourceId, ccmServiceAccountName)
				ingressFedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, ingressIdentityResourceId, ingressServiceAccountName)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &ccmFedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &oidcIssuer,
								Subject:   &ccmServiceAccountName,
							},
						},
					}, nil)
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ingressIdentityResourceId.ResourceGroup), gomock.Eq(ingressIdentityResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &ingressFedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &oidcIssuer,
								Subject:   &ccmServiceAccountName,
							},
						},
					}, nil)

				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Eq(ccmFedCredName), gomock.Any())
				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ingressIdentityResourceId.ResourceGroup), gomock.Eq(ingressIdentityResourceId.ResourceName), gomock.Eq(ingressFedCredName), gomock.Any())
			},
		},
		{
			name: "success - does not delete federated credentials that do not belong to the cluster",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				ccmFedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, ccmIdentityResourceId, ccmServiceAccountName)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &ccmFedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &oidcIssuer,
								Subject:   &ccmServiceAccountName,
							},
						},
						{
							Name: pointerutils.ToPtr("fedCredWithWrongAudience"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("something-else")},
								Issuer:    &oidcIssuer,
								Subject:   &ccmServiceAccountName,
							},
						},
						{
							Name: pointerutils.ToPtr("fedCredWithWrongIssuer"),
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    pointerutils.ToPtr("someOtherIssuer"),
								Subject:   &ccmServiceAccountName,
							},
						},
					}, nil)

				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Eq(ccmFedCredName), gomock.Any())
				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Eq("fedCredWithWrongAudience"), gomock.Any()).Times(0)
				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Eq("fedCredWithWrongIssuer"), gomock.Any()).Times(0)
			},
		},
		{
			name: "error - encounter blocking error deleting a federated credential",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-cluster",
								},
							},
						},
					},
				},
			},
			wantErr: "parsing failed for /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-cluster. Invalid resource Id format",
		},
		{
			name: "success - federated identity credentials client returns error when listing credentials but deletion continues",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Any()).
					Return(nil, fmt.Errorf("something unexpected occurred"))
			},
		},
		{
			name: "success - federated identity credentials client returns error when deleting credentials but deletion continues",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				ccmFedCredName := platformworkloadidentity.GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, ccmIdentityResourceId, ccmServiceAccountName)

				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Any()).
					Return([]*sdkmsi.FederatedIdentityCredential{
						{
							Name: &ccmFedCredName,
							Properties: &sdkmsi.FederatedIdentityCredentialProperties{
								Audiences: []*string{pointerutils.ToPtr("openshift")},
								Issuer:    &oidcIssuer,
								Subject:   &ccmServiceAccountName,
							},
						},
					}, nil)

				federatedIdentityCredentials.EXPECT().Delete(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Eq(ccmFedCredName), gomock.Any()).
					Return(sdkmsi.FederatedIdentityCredentialsClientDeleteResponse{}, fmt.Errorf("something unexpected occurred"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			federatedIdentityCredentials := mock_armmsi.NewMockFederatedIdentityCredentialsClient(controller)
			if tt.mocks != nil {
				tt.mocks(federatedIdentityCredentials)
			}

			m := manager{
				log:                                    logrus.NewEntry(logrus.StandardLogger()),
				doc:                                    tt.doc,
				clusterMsiFederatedIdentityCredentials: federatedIdentityCredentials,
			}

			err := m.deleteFederatedCredentials(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
