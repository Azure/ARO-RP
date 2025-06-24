package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_msidataplane "github.com/Azure/ARO-RP/pkg/util/mocks/msidataplane"
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

	nic := armnetwork.InterfacesClientGetResponse{
		Interface: armnetwork.Interface{
			Name:       &nicName,
			Location:   &location,
			ID:         &resourceId,
			Properties: &armnetwork.InterfacePropertiesFormat{},
		},
	}

	tests := []struct {
		name    string
		mocks   func(*mock_armnetwork.MockInterfacesClient)
		wantErr string
	}{
		{
			name: "nic is in succeeded provisioning state",
			mocks: func(armNetworkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic.Properties.ProvisioningState = pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded)
				armNetworkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(nic, nil)
				armNetworkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName, nil).Return(nil)
			},
		},
		{
			name: "nic is in failed provisioning state",
			mocks: func(armNetworkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic.Properties.ProvisioningState = pointerutils.ToPtr(armnetwork.ProvisioningStateFailed)
				armNetworkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(nic, nil)
				armNetworkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, nic.Interface, nil).Return(nil)
				armNetworkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName, nil).Return(nil)
			},
		},
		{
			name: "provisioning state is failed and CreateOrUpdateAndWait returns error",
			mocks: func(armNetworkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic.Properties.ProvisioningState = pointerutils.ToPtr(armnetwork.ProvisioningStateFailed)
				armNetworkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(nic, nil)
				armNetworkInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, nicName, nic.Interface, nil).Return(fmt.Errorf("Failed to update"))
			},
			wantErr: "Failed to update",
		},
		{
			name: "nic no longer exists - do nothing",
			mocks: func(armNetworkInterfaces *mock_armnetwork.MockInterfacesClient) {
				notFound := azcore.ResponseError{
					StatusCode: http.StatusNotFound,
				}
				armNetworkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(nic, &notFound)
			},
		},
		{
			name: "DeleteAndWait returns error",
			mocks: func(armNetworkInterfaces *mock_armnetwork.MockInterfacesClient) {
				nic.Properties.ProvisioningState = pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded)
				armNetworkInterfaces.EXPECT().Get(gomock.Any(), clusterRG, nicName, nil).Return(nic, nil)
				armNetworkInterfaces.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, nicName, nil).Return(fmt.Errorf("Failed to delete"))
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

			armNetworkInterfaces := mock_armnetwork.NewMockInterfacesClient(controller)

			tt.mocks(armNetworkInterfaces)

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
				armInterfaces: armNetworkInterfaces,
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
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: pointerutils.ToPtr("")},
			wantShouldDelete: false,
		},
		{
			name:             "resource group not managed by cluster",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: pointerutils.ToPtr("/somethingelse")},
			wantShouldDelete: false,
		},
		{
			name:             "resource group managed by cluster",
			getResourceGroup: mgmtfeatures.ResourceGroup{Name: &managedRGName, ManagedBy: pointerutils.ToPtr(clusterResourceId)},
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
			resourceGroups.EXPECT().DeleteAndWait(gomock.Any(), gomock.Eq(managedRGName)).Return(tt.deleteErr).Times(1)

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
						ID: pointerutils.ToPtr(nsgId),
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
						ID: pointerutils.ToPtr(nsgId),
						Properties: &armnetwork.SecurityGroupPropertiesFormat{
							Subnets: []*armnetwork.Subnet{
								{
									ID: pointerutils.ToPtr(subnetId),
								},
							},
						},
					},
				}
				securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(securityGroup, nil)
				subnets.EXPECT().Get(gomock.Any(), subnetId).Return(&mgmtnetwork.Subnet{
					ID: pointerutils.ToPtr(subnetId),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: pointerutils.ToPtr(nsgId),
						},
					},
				}, nil).Times(1)
				subnets.EXPECT().CreateOrUpdate(gomock.Any(), subnetId, &mgmtnetwork.Subnet{
					ID:                     pointerutils.ToPtr(subnetId),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{},
				}).Return(nil).Times(1)
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
		mocks   func(mockManager *mock_azsecrets.MockClient)
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
			mocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), secretName, nil).Return(azsecrets.DeleteSecretResponse{}, fmt.Errorf("error in DeleteSecret")).Times(1)
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
			mocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().DeleteSecret(gomock.Any(), secretName, nil).Return(azsecrets.DeleteSecretResponse{}, nil).Times(1)
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

			mockKvClient := mock_azsecrets.NewMockClient(controller)
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

	// cluster vars
	docID := "00000000-0000-0000-0000-000000000000"
	clusterID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/fakeResourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/fakeCluster"
	clusterResourceId, _ := azure.ParseResourceID(clusterID)
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	secretName := dataplane.ManagedIdentityCredentialsStoragePrefix + mockGuid
	identityIDPrefix := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/", mockGuid, clusterRGName)
	oidcIssuer := "https://fakeissuer.fakedomain/fakecluster"

	// service account vars
	ccmServiceAccountName := "system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"
	ccmIdentityResourceId, _ := azure.ParseResourceID(fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"))
	ingressServiceAccountName := "system:serviceaccount:openshift-ingress-operator:ingress-operator"
	ingressIdentityResourceId, _ := azure.ParseResourceID(fmt.Sprintf("%s/%s", identityIDPrefix, "cio"))

	// msi vars
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)
	placeholderString := "placeholder"
	placeholderTime := time.Now().Format(time.RFC3339)
	placeholderNotEligibleForRotationTime := time.Now().Add(-1 * time.Hour)
	placeholderEligibleForRotationTime := time.Now().Add(-1200 * time.Hour)
	placeholderCredentialsObject := &dataplane.ManagedIdentityCredentials{
		ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
			{
				ClientID:                   &placeholderString,
				ClientSecret:               &placeholderString,
				TenantID:                   &placeholderString,
				ResourceID:                 &miResourceId,
				AuthenticationEndpoint:     &placeholderString,
				CannotRenewAfter:           &placeholderTime,
				ClientSecretURL:            &placeholderString,
				MtlsAuthenticationEndpoint: &placeholderString,
				NotAfter:                   &placeholderTime,
				NotBefore:                  &placeholderTime,
				RenewAfter:                 &placeholderTime,
				CustomClaims: &dataplane.CustomClaims{
					XMSAzNwperimid: []string{placeholderString},
					XMSAzTm:        &placeholderString,
				},
				ObjectID: &placeholderString,
			},
		},
	}
	credentialsObjectBuffer, err := json.Marshal(placeholderCredentialsObject)
	if err != nil {
		panic(err)
	}
	credentialsObjectString := string(credentialsObjectBuffer)
	notEligibleForRotationResponse := azsecrets.GetSecretResponse{
		Secret: azsecrets.Secret{
			Value: &credentialsObjectString,
			Attributes: &azsecrets.SecretAttributes{
				NotBefore: &placeholderNotEligibleForRotationTime,
			},
		},
	}
	eligibleForRotationResponse := azsecrets.GetSecretResponse{
		Secret: azsecrets.Secret{
			Value: &credentialsObjectString,
			Attributes: &azsecrets.SecretAttributes{
				NotBefore: &placeholderEligibleForRotationTime,
			},
		},
	}

	tests := []struct {
		name             string
		doc              *api.OpenShiftClusterDocument
		mocks            func(*mock_armmsi.MockFederatedIdentityCredentialsClient)
		kvClientMocks    func(*mock_azsecrets.MockClient)
		msiDataplaneStub func(*mock_msidataplane.MockClient)
		wantErr          string
	}{
		{
			name: "success - cluster doc has nil PlatformWorkloadIdentities, MSI certificate valid",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
			},
		},
		{
			name: "success - cluster doc has nil PlatformWorkloadIdentities, MSI certificate eligible for rotation",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Version:    "4.14.40",
							OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
						IdentityURL: "https://foo.bar",
					},
				},
			},
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(placeholderCredentialsObject, nil)
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(eligibleForRotationResponse, nil).Times(1)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any(), nil).Return(azsecrets.SetSecretResponse{}, nil).Times(1)
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
							UpgradeableTo:              pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
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
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{},
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
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
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*sdkmsi.FederatedIdentityCredential{}, nil)
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
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
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-cluster",
								},
							},
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			wantErr: "parsing failed for /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-cluster. Invalid resource Id format",
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
			},
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			mocks: func(federatedIdentityCredentials *mock_armmsi.MockFederatedIdentityCredentialsClient) {
				federatedIdentityCredentials.EXPECT().List(gomock.Any(), gomock.Eq(ccmIdentityResourceId.ResourceGroup), gomock.Eq(ccmIdentityResourceId.ResourceName), gomock.Any()).
					Return(nil, fmt.Errorf("something unexpected occurred"))
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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
							UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", identityIDPrefix, "ccm"),
								},
							},
						},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(notEligibleForRotationResponse, nil).Times(1)
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

			mockKvClient := mock_azsecrets.NewMockClient(controller)
			if tt.kvClientMocks != nil {
				tt.kvClientMocks(mockKvClient)
			}

			factory := mock_msidataplane.NewMockClientFactory(controller)
			client := mock_msidataplane.NewMockClient(controller)
			if tt.msiDataplaneStub != nil {
				tt.msiDataplaneStub(client)
			}
			factory.EXPECT().NewClient(gomock.Any()).Return(client, nil).AnyTimes()

			m := manager{
				log:                                    logrus.NewEntry(logrus.StandardLogger()),
				doc:                                    tt.doc,
				clusterMsiFederatedIdentityCredentials: federatedIdentityCredentials,
				clusterMsiKeyVaultStore:                mockKvClient,
				msiDataplane:                           factory,
			}

			err := m.deleteFederatedCredentials(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
