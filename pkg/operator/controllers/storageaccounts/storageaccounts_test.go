package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/storage"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
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
			StorageSuffix:          storageSuffix,
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: strconv.FormatBool(operatorFlag),
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
			ProvisioningState: mgmtnetwork.Succeeded,
		})
	}
	return s
}

func TestReconcileManager(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name         string
		mocks        func(*mock_storage.MockAccountsClient, *mock_subnet.MockKubeManager, *mock_subnet.MockManager)
		instance     func(*arov1alpha1.Cluster)
		operatorFlag bool
		wantErr      error
	}{
		{
			name:         "Operator Flag enabled - nothing to do",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			name:         "Operator Flag enabled - all rules to all accounts because egress lockdown not enabled",
			operatorFlag: true,
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				// Service endpoints should be there if egress lockdown hasn't yet been enabled, but let's
				// pessimistically assume they have somehow been removed so we have the opportunity to
				// test a messy edge case.
				masterSubnet.ServiceEndpoints = nil
				workerSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			name:         "Operator flag and egress lockdown enabled - all rules to all accounts because storage service endpoint on subnets",
			operatorFlag: true,
			instance: func(cluster *arov1alpha1.Cluster) {
				cluster.Spec.GatewayDomains = []string{"somegatewaydomain.com"}
			},
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			name:         "Operator flag and egress lockdown enabled - worker subnet rule to all accounts because storage service endpoint on worker subnet",
			operatorFlag: true,
			instance: func(cluster *arov1alpha1.Cluster) {
				cluster.Spec.GatewayDomains = []string{"somegatewaydomain.com"}
			},
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				masterSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			name:         "Operator Flag and egress lockdown enabled - nothing to do because there are no service endpoints",
			operatorFlag: true,
			instance: func(cluster *arov1alpha1.Cluster) {
				cluster.Spec.GatewayDomains = []string{"somegatewaydomain.com"}
			},
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager, mgmtSubnet *mock_subnet.MockManager) {
				// Azure subnets
				masterSubnet := getValidSubnet(resourceIdMaster)
				workerSubnet := getValidSubnet(resourceIdWorker)

				masterSubnet.ServiceEndpoints = nil
				workerSubnet.ServiceEndpoints = nil

				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(masterSubnet, nil)
				mgmtSubnet.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(workerSubnet, nil)

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
			mgmtSubnet := mock_subnet.NewMockManager(controller)

			if tt.mocks != nil {
				tt.mocks(storage, kubeSubnet, mgmtSubnet)
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
				mgmtSubnets:    mgmtSubnet,
				kubeSubnets:    kubeSubnet,
				client:         clientFake,
			}

			err := r.reconcileAccounts(context.Background())
			if err != nil {
				if tt.wantErr == nil {
					t.Fatal(err)
				}
				if err.Error() != tt.wantErr.Error() || err == nil && tt.wantErr != nil {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.wantErr.Error(), err.Error(), tt.name)
				}
			}
		})
	}
}
