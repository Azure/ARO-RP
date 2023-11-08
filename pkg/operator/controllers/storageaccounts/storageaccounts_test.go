package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/storage"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	nsgv1MasterResourceId = managedResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + apisubnet.NSGControlPlaneSuffixV1
)

func getValidAccount(virtualNetworkResourceIDs []string) *mgmtstorage.Account {
	account := &mgmtstorage.Account{
		AccountProperties: &mgmtstorage.AccountProperties{
			NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
				VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{},
			},
		},
	}

	for _, rule := range virtualNetworkResourceIDs {
		*account.AccountProperties.NetworkRuleSet.VirtualNetworkRules = append(*account.AccountProperties.NetworkRuleSet.VirtualNetworkRules, mgmtstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: to.StringPtr(rule),
			Action:                   mgmtstorage.Allow,
		})
	}
	return account
}

func getValidSubnet(resourceId string) *mgmtnetwork.Subnet {
	s := &mgmtnetwork.Subnet{
		ID: to.StringPtr(resourceId),
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr(nsgv1MasterResourceId),
			},
			ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
	for _, endpoint := range api.SubnetsEndpoints {
		*s.SubnetPropertiesFormat.ServiceEndpoints = append(*s.SubnetPropertiesFormat.ServiceEndpoints, mgmtnetwork.ServiceEndpointPropertiesFormat{
			Service:           to.StringPtr(endpoint),
			Locations:         &[]string{location},
			ProvisioningState: mgmtnetwork.Succeeded,
		})
	}
	return s
}

func TestCheckClusterSubnetsToReconcile(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	additionalWorkerSubnetId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/workerAdditional"

	for _, tt := range []struct {
		name           string
		mocks          func(*mock_subnet.MockManager)
		clusterSubnets []string
		wantSubnets    []string
		wantErr        string
	}{
		{
			name: "no subnets have service endpoints - returns nothing",
			mocks: func(mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(masterSubnetId)
				workerSubnet := getValidSubnet(workerSubnetId)

				masterSubnet.ServiceEndpoints = nil
				workerSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), masterSubnetId).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), workerSubnetId).Return(workerSubnet, nil)
			},
			clusterSubnets: []string{masterSubnetId, workerSubnetId},
			wantSubnets:    []string{},
		},
		{
			name: "all subnets have service endpoints - returns all",
			mocks: func(mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(masterSubnetId)
				workerSubnet := getValidSubnet(workerSubnetId)

				mgmtSubnet.EXPECT().Get(gomock.Any(), masterSubnetId).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), workerSubnetId).Return(workerSubnet, nil)
			},
			clusterSubnets: []string{masterSubnetId, workerSubnetId},
			wantSubnets:    []string{masterSubnetId, workerSubnetId},
		},
		{
			name: "only worker subnet has service endpoint - returns only worker",
			mocks: func(mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(masterSubnetId)
				workerSubnet := getValidSubnet(workerSubnetId)

				masterSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), masterSubnetId).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), workerSubnetId).Return(workerSubnet, nil)
			},
			clusterSubnets: []string{masterSubnetId, workerSubnetId},
			wantSubnets:    []string{workerSubnetId},
		},
		{
			name: "additional worker subnet not found, do not return",
			mocks: func(mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(masterSubnetId)
				workerSubnet := getValidSubnet(workerSubnetId)

				masterSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), masterSubnetId).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), workerSubnetId).Return(workerSubnet, nil)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}
				mgmtSubnet.EXPECT().Get(gomock.Any(), additionalWorkerSubnetId).Return(nil, notFoundErr)
			},
			clusterSubnets: []string{masterSubnetId, workerSubnetId, additionalWorkerSubnetId},
			wantSubnets:    []string{workerSubnetId},
		},
		{
			name: "service endpoints exist but location does not match cluster",
			mocks: func(mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(masterSubnetId)
				workerSubnet := getValidSubnet(workerSubnetId)

				newMasterServiceEndpoints := []mgmtnetwork.ServiceEndpointPropertiesFormat{}

				for _, se := range *masterSubnet.ServiceEndpoints {
					se.Locations = &[]string{"not_a_real_place"}
					newMasterServiceEndpoints = append(newMasterServiceEndpoints, se)
				}

				masterSubnet.ServiceEndpoints = &newMasterServiceEndpoints

				newWorkerServiceEndpoints := []mgmtnetwork.ServiceEndpointPropertiesFormat{}

				for _, se := range *workerSubnet.ServiceEndpoints {
					se.Locations = &[]string{"not_a_real_place"}
					newWorkerServiceEndpoints = append(newWorkerServiceEndpoints, se)
				}

				workerSubnet.ServiceEndpoints = &newWorkerServiceEndpoints

				mgmtSubnet.EXPECT().Get(gomock.Any(), masterSubnetId).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), workerSubnetId).Return(workerSubnet, nil)
			},
			clusterSubnets: []string{masterSubnetId, workerSubnetId},
			wantSubnets:    []string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockSubnet := mock_subnet.NewMockManager(controller)

			if tt.mocks != nil {
				tt.mocks(mockSubnet)
			}

			r := reconcileManager{
				log: log,

				location:      location,
				resourceGroup: managedResourceGroupName,

				subnet: mockSubnet,
			}

			gotSubnets, err := r.checkClusterSubnetsToReconcile(context.Background(), tt.clusterSubnets)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if diff := cmp.Diff(tt.wantSubnets, gotSubnets, cmpoptsSortStringSlices); diff != "" {
				t.Errorf("wanted subnets %v but got %v, diff: %s", tt.wantSubnets, gotSubnets, diff)
			}
		})
	}
}

func TestReconcileAccounts(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name            string
		mocks           func(*mock_storage.MockAccountsClient)
		subnets         []string
		storageAccounts []string
		wantErr         string
	}{
		{
			name: "nothing to do",
			mocks: func(storage *mock_storage.MockAccountsClient) {
				// storage objects in azure
				result := getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId, masterSubnetId, workerSubnetId})
				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
			subnets:         []string{rpPeSubnetId, rpSubnetId, gwySubnetId, masterSubnetId, workerSubnetId},
			storageAccounts: []string{clusterStorageAccountName, registryStorageAccountName},
		},
		{
			name: "all rules to all accounts",
			mocks: func(storage *mock_storage.MockAccountsClient) {
				// storage objects in azure
				result := getValidAccount([]string{})
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId, masterSubnetId, workerSubnetId}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, updated)

				// we can't reuse these from above due to fact how gomock handles objects.
				// they are modified by the functions so they are not the same anymore
				result = getValidAccount([]string{})
				updated = mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId, masterSubnetId, workerSubnetId}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), managedResourceGroupName, registryStorageAccountName, updated)
			},
			subnets:         []string{rpPeSubnetId, rpSubnetId, gwySubnetId, masterSubnetId, workerSubnetId},
			storageAccounts: []string{clusterStorageAccountName, registryStorageAccountName},
		},
		{
			name: "worker subnet rule to all accounts because storage service endpoint on worker subnet",
			mocks: func(storage *mock_storage.MockAccountsClient) {
				// storage objects in azure
				result := getValidAccount([]string{})
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId, workerSubnetId}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, updated)

				// we can't reuse these from above due to fact how gomock handles objects.
				// they are modified by the functions so they are not the same anymore
				result = getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId})
				updated = mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId, workerSubnetId}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), managedResourceGroupName, registryStorageAccountName, updated)
			},
			subnets:         []string{rpPeSubnetId, rpSubnetId, gwySubnetId, workerSubnetId},
			storageAccounts: []string{clusterStorageAccountName, registryStorageAccountName},
		},
		{
			name: "nothing to do because no service endpoints",
			mocks: func(storage *mock_storage.MockAccountsClient) {
				// storage objects in azure
				result := getValidAccount([]string{rpPeSubnetId, rpSubnetId, gwySubnetId})
				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), managedResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
			subnets:         []string{rpPeSubnetId, rpSubnetId, gwySubnetId},
			storageAccounts: []string{clusterStorageAccountName, registryStorageAccountName},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockStorage := mock_storage.NewMockAccountsClient(controller)

			if tt.mocks != nil {
				tt.mocks(mockStorage)
			}

			r := reconcileManager{
				log: log,

				location:      location,
				resourceGroup: managedResourceGroupName,

				storage: mockStorage,
			}

			err := r.reconcileAccounts(context.Background(), tt.subnets, tt.storageAccounts)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
