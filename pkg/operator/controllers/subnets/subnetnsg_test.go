package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
	subscriptionId           = "0000000-0000-0000-0000-000000000000"
	clusterResourceGroupName = "aro-iljrzb5a"
	infraId                  = "abcd"
	clusterResourceGroupId   = "/subscriptions/" + subscriptionId + "/resourcegroups/" + clusterResourceGroupName
	vnetResourceGroup        = "vnet-rg"
	vnetName                 = "vnet"
	subnetNameWorker         = "worker"
	subnetNameMaster         = "master"

	nsgv1NodeResourceId   = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + subnet.NSGNodeSuffixV1
	nsgv1MasterResourceId = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + subnet.NSGControlPlaneSuffixV1
	nsgv2ResourceId       = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + subnet.NSGSuffixV2
)

func getValidClusterInstance() *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ArchitectureVersion:    0,
			ClusterResourceGroupID: clusterResourceGroupId,
			InfraID:                infraId,
		},
	}
}

func getValidSubnet() *mgmtnetwork.Subnet {
	return &mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr(nsgv1MasterResourceId),
			},
		},
	}
}

func getMachineObject(name, networkResourceGroup, vnet, subnet string, isMaster bool) (*machinev1beta1.Machine, error) {
	raw, err := azureproviderv1beta1.EncodeMachineSpec(&azureproviderv1beta1.AzureMachineProviderSpec{
		NetworkResourceGroup: networkResourceGroup,
		Vnet:                 vnet,
		Subnet:               subnet,
	})
	if err != nil {
		return nil, err
	}

	role := "worker"
	if isMaster {
		role = "master"
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineSetsNamespace,
			Labels: map[string]string{
				"machine.openshift.io/cluster-api-machine-role": role,
			},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: raw,
			},
		},
	}, nil
}

func TestReconcileManager(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name       string
		subnetMock func(*mock_subnet.MockManager, *mock_subnet.MockKubeManager)
		maocli     func() (*maofake.Clientset, error)
		instance   func(*arov1alpha1.Cluster)
		wantErr    error
	}{
		{
			name: "Architecture V1 - no change",
			maocli: func() (*maofake.Clientset, error) {
				machine1, err := getMachineObject("master", vnetResourceGroup, vnetName, subnetNameMaster, true)
				if err != nil {
					return nil, err
				}

				machine2, err := getMachineObject("worker", vnetResourceGroup, vnetName, subnetNameWorker, false)
				if err != nil {
					return nil, err
				}

				return maofake.NewSimpleClientset(machine1, machine2), nil
			},
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
				resourceIdMaster := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				resourceIdWorker := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker

				kmock.EXPECT().ListFromCluster(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(subnetObjectMaster, nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(subnetObjectWorker, nil)
			},
		},
		{
			name: "Architecture V1 - all fixup",
			maocli: func() (*maofake.Clientset, error) {
				machine1, err := getMachineObject("master", vnetResourceGroup, vnetName, subnetNameMaster, true)
				if err != nil {
					return nil, err
				}

				machine2, err := getMachineObject("worker", vnetResourceGroup, vnetName, subnetNameWorker, false)
				if err != nil {
					return nil, err
				}

				return maofake.NewSimpleClientset(machine1, machine2), nil
			},
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {

				resourceIdMaster := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				resourceIdWorker := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker

				kmock.EXPECT().ListFromCluster(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(subnetObjectMaster, nil)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
		},
		{
			name: "Architecture V1 - node only fixup",
			maocli: func() (*maofake.Clientset, error) {
				machine1, err := getMachineObject("master", vnetResourceGroup, vnetName, subnetNameMaster, true)
				if err != nil {
					return nil, err
				}

				machine2, err := getMachineObject("worker", vnetResourceGroup, vnetName, subnetNameWorker, false)
				if err != nil {
					return nil, err
				}

				return maofake.NewSimpleClientset(machine1, machine2), nil
			},
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {

				resourceIdMaster := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				resourceIdWorker := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker

				kmock.EXPECT().ListFromCluster(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(subnetObjectMaster, nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
		},
		{
			name: "Architecture V2 - no fixups",
			maocli: func() (*maofake.Clientset, error) {
				machine1, err := getMachineObject("master", vnetResourceGroup, vnetName, subnetNameMaster, true)
				if err != nil {
					return nil, err
				}

				machine2, err := getMachineObject("worker", vnetResourceGroup, vnetName, subnetNameWorker, false)
				if err != nil {
					return nil, err
				}

				return maofake.NewSimpleClientset(machine1, machine2), nil
			},
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {

				resourceIdMaster := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				resourceIdWorker := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker

				kmock.EXPECT().ListFromCluster(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(subnetObjectMaster, nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(subnetObjectWorker, nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = 1
			},
		},
		{
			name: "Architecture V2 - all nodes fixup",
			maocli: func() (*maofake.Clientset, error) {
				machine1, err := getMachineObject("master", vnetResourceGroup, vnetName, subnetNameMaster, true)
				if err != nil {
					return nil, err
				}

				machine2, err := getMachineObject("worker", vnetResourceGroup, vnetName, subnetNameWorker, false)
				if err != nil {
					return nil, err
				}

				return maofake.NewSimpleClientset(machine1, machine2), nil
			},
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {

				resourceIdMaster := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				resourceIdWorker := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker

				kmock.EXPECT().ListFromCluster(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: resourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(subnetObjectMaster, nil)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = 1
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnets := mock_subnet.NewMockManager(controller)
			kSubnets := mock_subnet.NewMockKubeManager(controller)
			if tt.subnetMock != nil {
				tt.subnetMock(subnets, kSubnets)
			}

			instance := getValidClusterInstance()
			if tt.instance != nil {
				tt.instance(instance)
			}

			r := reconcileManager{
				log:            log,
				instance:       instance,
				subscriptionID: subscriptionId,
				subnets:        subnets,
				kSubnets:       kSubnets,
			}

			err := r.reconcileSubnets(context.Background())
			if err != nil {
				if tt.wantErr == nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(tt.wantErr.Error(), err.Error()) {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.wantErr.Error(), err.Error(), tt.name)
				}
			}
			// we don't need to compare as mock should do the job

		})
	}
}
