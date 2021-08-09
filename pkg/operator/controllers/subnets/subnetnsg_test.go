package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mocksubnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
	subscriptionId = "0000000-0000-0000-0000-000000000000"
	vnet           = azure.Resource{
		SubscriptionID: subscriptionId,
		ResourceGroup:  "vnet-rg",
		ResourceName:   "vnet-name",
	}
	clusterResourceGroupName    = "aro-iljrzb5a"
	infraId                     = "abcd"
	clusterResourceGroupId      = "/subscriptions/" + subscriptionId + "/resourcegroups/" + clusterResourceGroupName
	vnetResourceGroup           = "vnet-rg"
	vnetResourceGroupResourceId = "/subscriptions/" + subscriptionId + "/resourcegroups/" + vnetResourceGroup
	vnetName                    = "vnet"
	subnetNameWorker            = "worker"
	subnetNameMaster            = "master"

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

func getValidWorkerMachine() *machinev1beta1.Machine {
	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker",
			Namespace: machineSetsNamespace,
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
		subnetMock func(*mocksubnet.MockManager)
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
			subnetMock: func(mock *mocksubnet.MockManager) {

				resourceId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectMaster, nil)

				resourceId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectWorker, nil)
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
			subnetMock: func(mock *mocksubnet.MockManager) {

				resourceId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectMaster, nil)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceId, subnetObjectMasterUpdate).Return(nil)

				resourceId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceId, subnetObjectWorkerUpdate).Return(nil)
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
			subnetMock: func(mock *mocksubnet.MockManager) {

				resourceId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectMaster, nil)

				resourceId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceId, subnetObjectWorkerUpdate).Return(nil)
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
			subnetMock: func(mock *mocksubnet.MockManager) {

				resourceId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectMaster, nil)

				resourceId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectWorker, nil)
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
			subnetMock: func(mock *mocksubnet.MockManager) {

				resourceId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectMaster, nil)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceId, subnetObjectMasterUpdate).Return(nil)

				resourceId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), resourceId).Return(subnetObjectWorker, nil)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), resourceId, subnetObjectWorkerUpdate).Return(nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = 1
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			manager := mocksubnet.NewMockManager(controller)
			if tt.subnetMock != nil {
				tt.subnetMock(manager)
			}

			instance := getValidClusterInstance()
			if tt.instance != nil {
				tt.instance(instance)
			}

			maocli, err := tt.maocli()
			if err != nil {
				t.Fatalf(err.Error())
			}

			r := reconcileManager{
				log:            log,
				maocli:         maocli,
				instance:       instance,
				subscriptionID: subscriptionId,
				manager:        manager,
			}

			err = r.reconcileSubnets(context.Background())
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

//func TestGetSubnets(t *testing.T) {
//	r := Reconciler{log: utillog.GetLogger()}
//	for _, tt := range []struct {
//		name             string
//		machinelabel     string
//		expectedMap      map[subnetDescriptor]bool
//		expectedMasterRG string
//		modify           func(*machinev1beta1.Machine, *machinev1beta1.Machine)
//		expectedErr      error
//	}{
//		{
//			name: "main path",
//			expectedMap: map[subnetDescriptor]bool{
//				{
//					resourceGroup: "netRG",
//					vnetName:      "workerVnet",
//					subnetName:    "workerSubnet",
//				}: false,
//				{
//					resourceGroup: "netRG",
//					vnetName:      "masterVnet",
//					subnetName:    "masterSubnet",
//				}: true,
//			},
//			expectedMasterRG: "masterRG",
//			modify:           func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {},
//		},
//		{
//			name:             "missing providerSpec",
//			expectedMap:      nil,
//			expectedMasterRG: "",
//			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
//				master.Spec.ProviderSpec.Value.Raw = []byte("")
//			},
//			expectedErr: fmt.Errorf("unexpected end of JSON input"),
//		},
//		{
//			name:             "missing master nodes",
//			expectedMap:      nil,
//			expectedMasterRG: "masterRG",
//			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
//				master.Labels = map[string]string{}
//			},
//			expectedErr: fmt.Errorf("master resource group not found"),
//		},
//	} {
//		t.Run(tt.name, func(t *testing.T) {
//			masterMachine := machinev1beta1.Machine{
//				ObjectMeta: metav1.ObjectMeta{
//					Name:      "master-0",
//					Namespace: "openshift-machine-api",
//					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
//				},
//				Spec: machinev1beta1.MachineSpec{
//					ProviderSpec: machinev1beta1.ProviderSpec{
//						Value: &runtime.RawExtension{
//							Raw: []byte("{\"resourceGroup\":\"masterRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"netRG\",\"vnet\":\"masterVnet\",\"subnet\":\"masterSubnet\"}"),
//						},
//					},
//				},
//			}
//			workerMachine := machinev1beta1.Machine{
//				ObjectMeta: metav1.ObjectMeta{
//					Name:      "worker-0",
//					Namespace: "openshift-machine-api",
//					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
//				},
//				Spec: machinev1beta1.MachineSpec{
//					ProviderSpec: machinev1beta1.ProviderSpec{
//						Value: &runtime.RawExtension{
//							Raw: []byte("{\"resourceGroup\":\"workerRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"netRG\",\"vnet\":\"workerVnet\",\"subnet\":\"workerSubnet\"}"),
//						},
//					},
//				},
//			}
//			tt.modify(&workerMachine, &masterMachine)
//			r.maocli = maofake.NewSimpleClientset(&workerMachine, &masterMachine)
//			subnetMap, masterRG, err := r.getSubnets(context.Background())
//			if err != nil {
//				if tt.expectedErr == nil {
//					t.Fatal(err)
//				}
//				if !strings.EqualFold(err.Error(), tt.expectedErr.Error()) {
//					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.expectedErr.Error(), err.Error(), tt.name)
//				}
//				return
//			}
//			if !strings.EqualFold(tt.expectedMasterRG, masterRG) {
//				t.Errorf("Expected Master Resource Group %s, got %s when processing %s testcase", tt.expectedMasterRG, masterRG, tt.name)
//			}
//			if tt.expectedMap != nil {
//				if len(tt.expectedMap) != len(subnetMap) {
//					t.Errorf("Expected Map length %d, doesn't match result map length %d when processing %s testcase", len(tt.expectedMap), len(subnetMap), tt.name)
//				}
//				for subnet := range tt.expectedMap {
//					value, present := subnetMap[subnet]
//					if !present {
//						t.Errorf("Subnet %s, %s, %s expected but not present in result when processing %s testcase", subnet.resourceGroup, subnet.vnetName, subnet.subnetName, tt.name)
//					}
//					if tt.expectedMap[subnet] != value {
//						t.Errorf("Value of isMaster boolean doesn't match for subnet %s, %s, %s when processing %s testcase", subnet.resourceGroup, subnet.vnetName, subnet.subnetName, tt.name)
//					}
//				}
//			}
//		})
//	}
//}
//
