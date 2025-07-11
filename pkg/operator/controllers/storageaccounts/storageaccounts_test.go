package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest"

	imageregistryv1 "github.com/openshift/api/imageregistry/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/storage"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
	location                 = "eastus"
	subscriptionId           = "0000000-0000-0000-0000-000000000000"
	clusterResourceGroupName = "aro-iljrzb5a"
	clusterResourceGroupId   = "/subscriptions/" + subscriptionId + "/resourcegroups/" + clusterResourceGroupName
	infraId                  = "abcd"
	vnetResourceGroup        = "vnet-rg"
	vnetName                 = "vnet"
	subnetNameWorker         = "worker"
	subnetNameMaster         = "master"
	nsgv1MasterResourceId    = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + apisubnet.NSGControlPlaneSuffixV1

	storageSuffix              = "random-suffix"
	clusterStorageAccountName  = "cluster" + storageSuffix
	registryStorageAccountName = "image-registry-account"

	resourceIdMaster = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
	resourceIdWorker = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
)

func getValidClusterInstance(operatorFlag bool) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ClusterResourceGroupID: clusterResourceGroupId,
			Location:               location,
			StorageSuffix:          storageSuffix,
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.StorageAccountsEnabled: strconv.FormatBool(operatorFlag),
			},
		},
	}
}

func getValidAccount(virtualNetworkResourceIDs []string) *mgmtstorage.Account {
	account := &mgmtstorage.Account{
		AccountProperties: &mgmtstorage.AccountProperties{
			NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
				VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{},
			},
		},
	}

	for _, rule := range virtualNetworkResourceIDs {
		*account.NetworkRuleSet.VirtualNetworkRules = append(*account.NetworkRuleSet.VirtualNetworkRules, mgmtstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: pointerutils.ToPtr(rule),
			Action:                   mgmtstorage.ActionAllow,
		})
	}
	return account
}

func getValidSubnet(resourceId string) *armnetwork.Subnet {
	s := &armnetwork.Subnet{
		ID: pointerutils.ToPtr(resourceId),
		Properties: &armnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: pointerutils.ToPtr(nsgv1MasterResourceId),
			},
			ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
	for _, endpoint := range api.SubnetsEndpoints {
		se := &armnetwork.ServiceEndpointPropertiesFormat{
			Service:           pointerutils.ToPtr(endpoint),
			Locations:         []*string{pointerutils.ToPtr(location)},
			ProvisioningState: (*armnetwork.ProvisioningState)(pointerutils.ToPtr(string(armnetwork.ProvisioningStateSucceeded))),
		}
		s.Properties.ServiceEndpoints = append(s.Properties.ServiceEndpoints, se)
	}
	return s
}

func TestReconcileManager(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name         string
		mocks        func(*mock_storage.MockAccountsClient, *mock_subnet.MockKubeManager, *mock_armnetwork.MockSubnetsClient)
		instance     func(*arov1alpha1.Cluster)
		operatorFlag bool
		wantErr      error
	}{
		{
			name:         "Operator Flag enabled - nothing to do",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{resourceIdMaster, resourceIdWorker})
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
		},
		{
			name:         "Operator Flag disabled - nothing to do",
			operatorFlag: false,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{resourceIdMaster, resourceIdWorker})
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
		},
		{
			name:         "Operator Flag enabled - all rules to all accounts",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{})
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster, resourceIdWorker}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, updated)

				// we can't reuse these from above due to fact how gomock handles objects.
				// they are modified by the functions so they are not the same anymore
				result = getValidAccount([]string{})
				updated = mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster, resourceIdWorker}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, updated)
			},
		},
		{
			name:         "Operator Flag enabled - not found error on getting worker subnet skips subnet",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{}, notFoundErr)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{})
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, updated)

				// we can't reuse these from above due to fact how gomock handles objects.
				// they are modified by the functions so they are not the same anymore
				result = getValidAccount([]string{})
				updated = mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, updated)
			},
		},
		{
			name:         "Operator flag enabled - worker subnet rule to all accounts because storage service endpoint on worker subnet",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				masterSubnet.Properties.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{})
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdWorker}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, updated)

				// we can't reuse these from above due to fact how gomock handles objects.
				// they are modified by the functions so they are not the same anymore
				result = getValidAccount([]string{})
				updated = mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdWorker}).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, updated)
			},
		},
		{
			name:         "Operator flag enabled - nothing to do because no service endpoints",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				masterSubnet.Properties.ServiceEndpoints = nil
				workerSubnet.Properties.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{})
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
		},
		{
			name:         "Operator flag enabled - nothing to do because the storage endpoint is there but the location does not match the cluster",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_armnetwork.MockSubnetsClient) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				newMasterServiceEndpoints := []*armnetwork.ServiceEndpointPropertiesFormat{}

				for _, se := range masterSubnet.Properties.ServiceEndpoints {
					se.Locations = []*string{pointerutils.ToPtr("not_a_real_place")}
					newMasterServiceEndpoints = append(newMasterServiceEndpoints, se)
				}

				masterSubnet.Properties.ServiceEndpoints = newMasterServiceEndpoints

				newWorkerServiceEndpoints := []*armnetwork.ServiceEndpointPropertiesFormat{}

				for _, se := range workerSubnet.Properties.ServiceEndpoints {
					se.Locations = []*string{pointerutils.ToPtr("not_a_real_place")}
					newWorkerServiceEndpoints = append(newWorkerServiceEndpoints, se)
				}

				workerSubnet.Properties.ServiceEndpoints = newWorkerServiceEndpoints

				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *masterSubnet}, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *workerSubnet}, nil)

				// cluster subnets
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// storage objects in azure
				result := getValidAccount([]string{})
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).Return(*result, nil)
				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, gomock.Any()).Return(*result, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			storage := mock_storage.NewMockAccountsClient(controller)
			kubeSubnet := mock_subnet.NewMockKubeManager(controller)
			subnet := mock_armnetwork.NewMockSubnetsClient(controller)

			if tt.mocks != nil {
				tt.mocks(storage, kubeSubnet, subnet)
			}

			instance := getValidClusterInstance(tt.operatorFlag)
			if tt.instance != nil {
				tt.instance(instance)
			}

			rc := &imageregistryv1.Config{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: imageregistryv1.ImageRegistrySpec{
					Storage: imageregistryv1.ImageRegistryConfigStorage{
						Azure: &imageregistryv1.ImageRegistryConfigStorageAzure{
							AccountName: registryStorageAccountName,
						},
					},
				},
			}
			clientFake := fake.NewClientBuilder().WithObjects(rc).Build()

			r := reconcileManager{
				log:            log,
				instance:       instance,
				subscriptionID: subscriptionId,
				storage:        storage,
				subnets:        subnet,
				kubeSubnets:    kubeSubnet,
				client:         clientFake,
			}

			err := r.reconcileAccounts(context.Background())
			if err != nil {
				if tt.wantErr == nil {
					t.Fatal(err)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.wantErr.Error(), err.Error(), tt.name)
				}
			}
		})
	}
}
