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
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
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
		mocks   func(*mock_armnetwork.MockSecurityGroupsClient, *mock_armnetwork.MockSubnetsClient)
		wantErr string
	}{
		{
			name: "empty security group",
			mocks: func(securityGroups *mock_armnetwork.MockSecurityGroupsClient, subnets *mock_armnetwork.MockSubnetsClient) {
				securityGroup := armnetwork.SecurityGroupsClientGetResponse{
					SecurityGroup: armnetwork.SecurityGroup{
						ID: pointerutils.ToPtr(nsgId),
						Properties: &armnetwork.SecurityGroupPropertiesFormat{
							Subnets: []*armnetwork.Subnet{},
						},
					},
				}
				securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(securityGroup, nil)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), nil).Times(0)
			},
		},
		{
			name: "fails to parse subnet ID",
			mocks: func(securityGroups *mock_armnetwork.MockSecurityGroupsClient, subnets *mock_armnetwork.MockSubnetsClient) {
				invalidSubnetId := "invalid-subnet-id"
				securityGroup := armnetwork.SecurityGroupsClientGetResponse{
					SecurityGroup: armnetwork.SecurityGroup{
						ID: pointerutils.ToPtr(nsgId),
						Properties: &armnetwork.SecurityGroupPropertiesFormat{
							Subnets: []*armnetwork.Subnet{
								{
									ID: pointerutils.ToPtr(invalidSubnetId),
								},
							},
						},
					},
				}
				securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(securityGroup, nil)
				// Should not call subnets.Get or CreateOrUpdateAndWait due to parse error
				subnets.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), nil).Times(0)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), nil).Times(0)
			},
			wantErr: "400: InvalidResourceID: invalid-subnet-id: Invalid subnet resource ID format. For more details, please refer to https://docs.microsoft.com/azure/azure-resource-manager/management/resource-name-rules",
		},
		{
			name: "disconnects subnets",
			mocks: func(securityGroups *mock_armnetwork.MockSecurityGroupsClient, subnets *mock_armnetwork.MockSubnetsClient) {
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
				subnets.EXPECT().Get(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						ID: pointerutils.ToPtr(subnetId),
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: pointerutils.ToPtr(nsgId),
							},
						},
					},
				}, nil).Times(1)
				subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", armnetwork.Subnet{
					ID: pointerutils.ToPtr(subnetId),
					Properties: &armnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: nil,
					},
				}, nil).Return(nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			securityGroups := mock_armnetwork.NewMockSecurityGroupsClient(controller)
			subnets := mock_armnetwork.NewMockSubnetsClient(controller)

			tt.mocks(securityGroups, subnets)

			m := manager{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				armSecurityGroups: securityGroups,
				armSubnets:        subnets,
			}

			ctx := context.Background()
			err := m.disconnectSecurityGroup(ctx, nsgId)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantRetry  bool
		wantLogMsg string
	}{
		{
			name:       "retryable 429 returns true and logs",
			err:        autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
			wantRetry:  true,
			wantLogMsg: "transient error on test-op, retrying:",
		},
		{
			name:      "non-retryable error returns false",
			err:       errors.New("permanent failure"),
			wantRetry: false,
		},
		{
			name:       "retryable azcore 429 returns true and logs",
			err:        &azcore.ResponseError{StatusCode: http.StatusTooManyRequests},
			wantRetry:  true,
			wantLogMsg: "transient error on test-op, retrying:",
		},
		{
			name: "retryable autorest 409+Retry-After returns true and logs",
			err: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Response: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
			wantRetry:  true,
			wantLogMsg: "transient error on test-op, retrying:",
		},
		{
			name: "retryable azcore 409+Retry-After returns true and logs",
			err: &azcore.ResponseError{
				StatusCode: http.StatusConflict,
				RawResponse: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
			wantRetry:  true,
			wantLogMsg: "transient error on test-op, retrying:",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			logger.SetLevel(logrus.WarnLevel)

			m := &manager{log: logrus.NewEntry(logger)}
			predicate := m.isRetryable("test-op")

			got := predicate(tt.err)
			assert.Equal(t, tt.wantRetry, got)

			if tt.wantLogMsg != "" {
				require.Len(t, hook.Entries, 1)
				assert.Contains(t, hook.LastEntry().Message, tt.wantLogMsg)
				assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
			} else {
				assert.Empty(t, hook.Entries)
			}
		})
	}
}

func TestDisconnectSecurityGroupRetry(t *testing.T) {
	subscription := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "test-rg"
	nsgID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/test-nsg", subscription, resourceGroup)
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscription, resourceGroup)

	for _, tt := range []struct {
		name     string
		firstErr error
	}{
		{
			name:     "retry on 429 (autorest): succeeds on second attempt",
			firstErr: autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
		},
		{
			name:     "retry on 409 Please retry later (autorest): succeeds on second attempt",
			firstErr: autorest.DetailedError{StatusCode: http.StatusConflict, Original: errors.New("ConflictingConcurrentWriteNotAllowed: Please retry later.")},
		},
		{
			name:     "retry on 429 (azcore): succeeds on second attempt",
			firstErr: &azcore.ResponseError{StatusCode: http.StatusTooManyRequests},
		},
		{
			name: "retry on azcore 409+Retry-After: succeeds on second attempt",
			firstErr: &azcore.ResponseError{
				StatusCode: http.StatusConflict,
				RawResponse: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
		},
		{
			name: "retry on autorest 409+Retry-After: succeeds on second attempt",
			firstErr: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Response: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			origBackoff := transientRetryBackoff
			transientRetryBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
			defer func() { transientRetryBackoff = origBackoff }()

			controller := gomock.NewController(t)
			defer controller.Finish()

			securityGroups := mock_armnetwork.NewMockSecurityGroupsClient(controller)
			subnets := mock_armnetwork.NewMockSubnetsClient(controller)

			securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(armnetwork.SecurityGroupsClientGetResponse{
				SecurityGroup: armnetwork.SecurityGroup{
					ID: pointerutils.ToPtr(nsgID),
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						Subnets: []*armnetwork.Subnet{{ID: pointerutils.ToPtr(subnetID)}},
					},
				},
			}, nil)
			subnets.EXPECT().Get(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", nil).Return(armnetwork.SubnetsClientGetResponse{
				Subnet: armnetwork.Subnet{
					ID: pointerutils.ToPtr(subnetID),
					Properties: &armnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &armnetwork.SecurityGroup{ID: pointerutils.ToPtr(nsgID)},
					},
				},
			}, nil)
			first := subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", gomock.Any(), nil).Return(tt.firstErr)
			subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", gomock.Any(), nil).Return(nil).After(first)

			m := manager{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				armSecurityGroups: securityGroups,
				armSubnets:        subnets,
			}

			assert.NoError(t, m.disconnectSecurityGroup(context.Background(), nsgID))
		})
	}
}

// TestDisconnectSecurityGroupRetryExhausted verifies that retry exhaustion propagates the error.
// Uses a single representative error (autorest 429); the exhaustion path is the same for all retryable errors.
func TestDisconnectSecurityGroupRetryExhausted(t *testing.T) {
	origBackoff := transientRetryBackoff
	transientRetryBackoff = wait.Backoff{Steps: 1, Duration: time.Millisecond, Factor: 2.0} // Steps: 1 = 1 attempt, 0 retries
	defer func() { transientRetryBackoff = origBackoff }()

	subscription := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "test-rg"
	nsgID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/test-nsg", subscription, resourceGroup)
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscription, resourceGroup)

	controller := gomock.NewController(t)
	defer controller.Finish()

	securityGroups := mock_armnetwork.NewMockSecurityGroupsClient(controller)
	subnets := mock_armnetwork.NewMockSubnetsClient(controller)

	securityGroups.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(armnetwork.SecurityGroupsClientGetResponse{
		SecurityGroup: armnetwork.SecurityGroup{
			ID: pointerutils.ToPtr(nsgID),
			Properties: &armnetwork.SecurityGroupPropertiesFormat{
				Subnets: []*armnetwork.Subnet{{ID: pointerutils.ToPtr(subnetID)}},
			},
		},
	}, nil)
	subnets.EXPECT().Get(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", nil).Return(armnetwork.SubnetsClientGetResponse{
		Subnet: armnetwork.Subnet{
			ID: pointerutils.ToPtr(subnetID),
			Properties: &armnetwork.SubnetPropertiesFormat{
				NetworkSecurityGroup: &armnetwork.SecurityGroup{ID: pointerutils.ToPtr(nsgID)},
			},
		},
	}, nil)
	subnets.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, "test-vnet", "test-subnet", gomock.Any(), nil).Return(
		autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
	)

	m := manager{
		log:               logrus.NewEntry(logrus.StandardLogger()),
		armSecurityGroups: securityGroups,
		armSubnets:        subnets,
	}

	err := m.disconnectSecurityGroup(context.Background(), nsgID)
	require.Error(t, err)
	var cloudErr *api.CloudError
	assert.ErrorAs(t, err, &cloudErr, "expected *api.CloudError wrapping the exhausted retry error")
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
	now := func() time.Time {
		return time.Date(2025, time.September, 29, 16, 0, 0, 0, time.UTC)
	}
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
			Value:      &credentialsObjectString,
			Attributes: &azsecrets.SecretAttributes{},
			Tags: map[string]*string{
				dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now().Add(1 * time.Hour).Format(time.RFC3339)),
				dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now().Add(2 * time.Hour).Format(time.RFC3339)),
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
			name: "success - cluster doc has nil PlatformWorkloadIdentities, exit early",
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
		},
		{
			name: "success - cluster doc has no oidc issuer, exit early",
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
		{
			name: "success - ensureClusterMsiCertificate fails but deletion continues",
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(azsecrets.GetSecretResponse{}, fmt.Errorf("key vault error")).Times(1)
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

			mockEnv := mock_env.NewMockInterface(controller)
			mockEnv.EXPECT().Now().AnyTimes().DoAndReturn(now)

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
				env:                                    mockEnv,
			}

			err := m.deleteFederatedCredentials(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

// doneFutureAPI is a minimal azure.FutureAPI implementation that reports immediate completion.
// Used in tests to avoid actual HTTP polling on azure.Future.
type doneFutureAPI struct{}

func (d *doneFutureAPI) Response() *http.Response               { return nil }
func (d *doneFutureAPI) Status() string                         { return "Succeeded" }
func (d *doneFutureAPI) PollingMethod() azure.PollingMethodType { return azure.PollingUnknown }
func (d *doneFutureAPI) DoneWithContext(context.Context, autorest.Sender) (bool, error) {
	return true, nil
}
func (d *doneFutureAPI) GetPollingDelay() (time.Duration, bool)                      { return 0, false }
func (d *doneFutureAPI) WaitForCompletionRef(context.Context, autorest.Client) error { return nil }
func (d *doneFutureAPI) MarshalJSON() ([]byte, error)                                { return []byte(`{}`), nil }
func (d *doneFutureAPI) UnmarshalJSON([]byte) error                                  { return nil }
func (d *doneFutureAPI) PollingURL() string                                          { return "" }
func (d *doneFutureAPI) GetResult(autorest.Sender) (*http.Response, error)           { return nil, nil }

func TestDeleteResourcesRetry(t *testing.T) {
	subscription := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "test-rg"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/test-pip", subscription, resourceGroup)

	for _, tt := range []struct {
		name     string
		firstErr error
	}{
		{
			name:     "retry on 429 (autorest): succeeds on second attempt",
			firstErr: autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
		},
		{
			name:     "retry on 409 Please retry later (autorest): succeeds on second attempt",
			firstErr: autorest.DetailedError{StatusCode: http.StatusConflict, Original: errors.New("ConflictingConcurrentWriteNotAllowed: Please retry later.")},
		},
		{
			name:     "retry on 429 (azcore): succeeds on second attempt",
			firstErr: &azcore.ResponseError{StatusCode: http.StatusTooManyRequests},
		},
		{
			name: "retry on azcore 409+Retry-After: succeeds on second attempt",
			firstErr: &azcore.ResponseError{
				StatusCode: http.StatusConflict,
				RawResponse: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
		},
		{
			name: "retry on autorest 409+Retry-After: succeeds on second attempt",
			firstErr: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Response: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			origBackoff := transientRetryBackoff
			transientRetryBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
			defer func() { transientRetryBackoff = origBackoff }()

			controller := gomock.NewController(t)
			defer controller.Finish()

			resources := mock_features.NewMockResourcesClient(controller)
			resources.EXPECT().ListByResourceGroup(gomock.Any(), resourceGroup, "", "", nil).Return(
				[]mgmtfeatures.GenericResourceExpanded{
					{
						ID:   pointerutils.ToPtr(resourceID),
						Type: pointerutils.ToPtr("Microsoft.Network/publicIPAddresses"),
					},
				}, nil,
			)
			first := resources.EXPECT().DeleteByID(gomock.Any(), resourceID, gomock.Any()).Return(
				mgmtfeatures.ResourcesDeleteByIDFuture{}, tt.firstErr,
			)
			resources.EXPECT().DeleteByID(gomock.Any(), resourceID, gomock.Any()).Return(
				mgmtfeatures.ResourcesDeleteByIDFuture{FutureAPI: &doneFutureAPI{}}, nil,
			).After(first)
			resources.EXPECT().Client().Return(autorest.Client{})

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, resourceGroup),
							},
						},
					},
				},
				resources: resources,
			}

			assert.NoError(t, m.deleteResources(context.Background()))
		})
	}
}

// TestDeleteResourcesRetryExhausted verifies that retry exhaustion propagates the error.
// Uses a single representative error (autorest 429); the exhaustion path is the same for all retryable errors.
func TestDeleteResourcesRetryExhausted(t *testing.T) {
	origBackoff := transientRetryBackoff
	transientRetryBackoff = wait.Backoff{Steps: 1, Duration: time.Millisecond, Factor: 2.0} // Steps: 1 = 1 attempt, 0 retries
	defer func() { transientRetryBackoff = origBackoff }()

	subscription := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "test-rg"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/test-pip", subscription, resourceGroup)

	controller := gomock.NewController(t)
	defer controller.Finish()

	resources := mock_features.NewMockResourcesClient(controller)
	resources.EXPECT().ListByResourceGroup(gomock.Any(), resourceGroup, "", "", nil).Return(
		[]mgmtfeatures.GenericResourceExpanded{
			{
				ID:   pointerutils.ToPtr(resourceID),
				Type: pointerutils.ToPtr("Microsoft.Network/publicIPAddresses"),
			},
		}, nil,
	)
	resources.EXPECT().DeleteByID(gomock.Any(), resourceID, gomock.Any()).Return(
		mgmtfeatures.ResourcesDeleteByIDFuture{}, autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
	)

	m := manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, resourceGroup),
					},
				},
			},
		},
		resources: resources,
	}

	err := m.deleteResources(context.Background())
	require.Error(t, err)
	var cloudErr *api.CloudError
	assert.NotErrorAs(t, err, &cloudErr, "expected raw error, not *api.CloudError: nil Original passes through deleteByIdCloudError unchanged")
}

func TestDeleteByIdCloudError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantErr    string
		wantStatus int
	}{
		{
			name: "CannotDeleteLoadBalancerWithPrivateLinkService",
			err: autorest.DetailedError{
				Original: errors.New(`Code="CannotDeleteLoadBalancerWithPrivateLinkService"`),
			},
			wantErr:    `400: CannotDeleteLoadBalancerWithPrivateLinkService: features.ResourcesClient#DeleteByID: Code="CannotDeleteLoadBalancerWithPrivateLinkService"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "AuthorizationFailed",
			err: autorest.DetailedError{
				Original: errors.New(`Code="AuthorizationFailed"`),
			},
			wantErr:    `403: Forbidden: features.ResourcesClient#DeleteByID: Code="AuthorizationFailed"`,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "InUseSubnetCannotBeDeleted",
			err: autorest.DetailedError{
				Original: errors.New(`Code="InUseSubnetCannotBeDeleted"`),
			},
			wantErr:    `400: InUseSubnetCannotBeDeleted: features.ResourcesClient#DeleteByID: Code="InUseSubnetCannotBeDeleted"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "ScopeLocked",
			err: autorest.DetailedError{
				Original: errors.New(`Code="ScopeLocked"`),
			},
			wantErr:    `409: ScopeLocked: features.ResourcesClient#DeleteByID: Code="ScopeLocked"`,
			wantStatus: http.StatusConflict,
		},
		{
			name: "unrecognized autorest error passes through",
			err: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Original:   errors.New("something else"),
			},
			wantErr: `#: : StatusCode=409 -- Original Error: something else`,
		},
		{
			name:    "non-autorest error passes through",
			err:     errors.New("generic error"),
			wantErr: "generic error",
		},
		{
			name:    "autorest DetailedError with nil Original passes through",
			err:     autorest.DetailedError{StatusCode: http.StatusConflict},
			wantErr: "#: : StatusCode=409",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteByIdCloudError(tt.err)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			var cloudErr *api.CloudError
			if tt.wantStatus != 0 {
				require.ErrorAs(t, err, &cloudErr, "expected *api.CloudError, got %T", err)
				assert.Equal(t, tt.wantStatus, cloudErr.StatusCode)
			} else {
				assert.NotErrorAs(t, err, &cloudErr, "expected non-CloudError, got %T", err)
			}
		})
	}
}
