package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/golang/mock/gomock"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	imageregistryfake "github.com/openshift/client-go/imageregistry/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/storage"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func TestReconcile(t *testing.T) {
	imagecli := imageregistryfake.NewSimpleClientset(
		&imageregistryv1.Config{
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
		},
	)

	instance := &arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ClusterResourceGroupID: clusterResourceGroupId,
			ServiceSubnets: []string{
				rpServiceSubnet,
				gatewayServiceSubnet,
			},
			StorageSuffix: storageSuffix,
		},
	}

	for _, tt := range []struct {
		name              string
		mocks             func(*mock_storage.MockAccountsClient, *mock_subnet.MockKubeManager)
		imageregistrycli  imageregistryclient.Interface
		controllerManaged bool
		wantErr           string
	}{
		{
			name: "kubesubnets returns error",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return(nil, errors.New("failed to list kube subnets"))
			},
			wantErr: "failed to list kube subnets",
		},
		{
			name: "ImageRegistry configs not found",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{{ResourceID: resourceIdMaster}, {ResourceID: resourceIdWorker}}, nil)
			},
			// Empty imageregistrycli on purpose for not found
			imageregistrycli: imageregistryfake.NewSimpleClientset(),
			wantErr:          `configs.imageregistry.operator.openshift.io "cluster" not found`,
		},
		{
			name: "managed == false; cannot fetch storage properties",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{{ResourceID: resourceIdMaster}, {ResourceID: resourceIdWorker}}, nil)
				storage.EXPECT().
					GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, mgmtstorage.AccountExpand("")).
					Return(mgmtstorage.Account{}, errors.New("failed to get properties on storage account"))
			},
			imageregistrycli:  imagecli,
			controllerManaged: false,
			wantErr:           `failed to get properties on storage account`,
		},
		{
			name: "managed == false; storage account update returns error",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{{ResourceID: resourceIdMaster}, {ResourceID: resourceIdWorker}}, nil)

				result := getValidAccount([]string{}, mgmtstorage.DefaultActionDeny)
				storage.EXPECT().
					GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, mgmtstorage.AccountExpand("")).
					Return(result, nil)

				storage.EXPECT().
					Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, gomock.Any()).
					Return(mgmtstorage.Account{}, errors.New("failed to update storage account"))
			},
			imageregistrycli:  imagecli,
			controllerManaged: false,
			wantErr:           `failed to update storage account`,
		},
		{
			name: "managed == false; reconcile default action and missing subnets",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{{ResourceID: resourceIdMaster}, {ResourceID: resourceIdWorker}}, nil)

				// storage objects in azure
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster, resourceIdWorker, rpServiceSubnet, gatewayServiceSubnet}, mgmtstorage.DefaultActionAllow).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, mgmtstorage.AccountExpand("")).Return(getValidAccount([]string{}, mgmtstorage.DefaultActionDeny), nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, AccountUpdateParamsEq(updated))

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, mgmtstorage.AccountExpand("")).Return(getValidAccount([]string{}, mgmtstorage.DefaultActionDeny), nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, AccountUpdateParamsEq(updated))

			},
			imageregistrycli:  imagecli,
			controllerManaged: false,
			wantErr:           "",
		},
		{
			name: "managed == true; reconcile default action and missing subnets",
			mocks: func(storage *mock_storage.MockAccountsClient, kubeSubnet *mock_subnet.MockKubeManager) {
				kubeSubnet.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{{ResourceID: resourceIdMaster}, {ResourceID: resourceIdWorker}}, nil)

				// storage objects in azure
				updated := mgmtstorage.AccountUpdateParameters{
					AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
						NetworkRuleSet: getValidAccount([]string{resourceIdMaster, resourceIdWorker, rpServiceSubnet, gatewayServiceSubnet}, mgmtstorage.DefaultActionDeny).NetworkRuleSet,
					},
				}

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, mgmtstorage.AccountExpand("")).Return(getValidAccount([]string{}, mgmtstorage.DefaultActionAllow), nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, clusterStorageAccountName, AccountUpdateParamsEq(updated))

				storage.EXPECT().GetProperties(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, mgmtstorage.AccountExpand("")).Return(getValidAccount([]string{}, mgmtstorage.DefaultActionAllow), nil)
				storage.EXPECT().Update(gomock.Any(), clusterResourceGroupName, registryStorageAccountName, AccountUpdateParamsEq(updated))

			},
			imageregistrycli:  imagecli,
			controllerManaged: true,
			wantErr:           "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			storage := mock_storage.NewMockAccountsClient(controller)
			kubeSubnet := mock_subnet.NewMockKubeManager(controller)

			if tt.mocks != nil {
				tt.mocks(storage, kubeSubnet)
			}

			r := reconcileManager{
				instance: instance,

				imageregistrycli: tt.imageregistrycli,
				kubeSubnets:      kubeSubnet,
				storage:          storage,
			}

			err := r.reconcileAccounts(context.Background(), tt.controllerManaged)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}
