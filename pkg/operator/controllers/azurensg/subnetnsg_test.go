package azurensg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

var (
	vnet = azure.Resource{
		SubscriptionID: "0000000-0000-0000-0000-000000000000",
		ResourceGroup:  "vnet-rg",
		ResourceName:   "vnet-name",
	}
)

func TestEnsureSubnetNSG(t *testing.T) {
	r := Reconciler{log: utillog.GetLogger()}
	for _, tt := range []struct {
		name                string
		nsgname             string
		architectureVersion api.ArchitectureVersion
		expectedValue       string
		modifySubnet        func(*mgmtnetwork.Subnet)
		expectedErr         error
		expectUpdate        bool
	}{
		{
			name:                "mismatched NSG in Architecture v1",
			nsgname:             "someotherv1-nsg",
			architectureVersion: api.ArchitectureVersionV1,
			expectedValue:       "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/resourcegroup/providers/Microsoft.Network/networkSecurityGroups/infraid-node-nsg",
			expectUpdate:        true,
		},
		{
			name:                "correct NSG in Architecture v2",
			nsgname:             "infraid-nsg",
			architectureVersion: api.ArchitectureVersionV2,
			expectedValue:       "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/resourcegroup/providers/Microsoft.Network/networkSecurityGroups/infraid-nsg",
			expectUpdate:        false,
		},
		{
			name:                "mismatched NSG in Architecture v2",
			nsgname:             "someotherv2-nsg",
			architectureVersion: api.ArchitectureVersionV2,
			expectedValue:       "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/resourcegroup/providers/Microsoft.Network/networkSecurityGroups/infraid-nsg",
			expectUpdate:        true,
		},
		{
			name:                "missing fields",
			nsgname:             "someotherv2-nsg",
			architectureVersion: api.ArchitectureVersionV2,
			expectedValue:       "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/resourcegroup/providers/Microsoft.Network/networkSecurityGroups/infraid-nsg",
			modifySubnet: func(subnet *mgmtnetwork.Subnet) {
				subnet.SubnetPropertiesFormat = nil
			},
			expectedErr:  fmt.Errorf("received nil, expected a value in SubnetProperties when trying to Get subnet vnet-name/subnet in resource group vnet-rg"),
			expectUpdate: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			fakeSubnet := mgmtnetwork.Subnet{SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/resourcegroup/providers/Microsoft.Network/networkSecurityGroups/" + tt.nsgname),
				},
			}}
			if tt.modifySubnet != nil {
				tt.modifySubnet(&fakeSubnet)
			}
			subnetsClient := mock_network.NewMockSubnetsClient(controller)
			subnetsClient.EXPECT().Get(context.Background(), "vnet-rg", "vnet-name", "subnet", "").
				Return(fakeSubnet, nil)
			if tt.expectUpdate {
				subnetsClient.EXPECT().CreateOrUpdateAndWait(context.Background(), "vnet-rg", "vnet-name", "subnet", fakeSubnet).
					Return(nil)
			}
			err := r.ensureSubnetNSG(context.Background(), subnetsClient, vnet.SubscriptionID, "resourcegroup", "infraid", tt.architectureVersion, vnet.ResourceGroup, vnet.ResourceName, "subnet", true)
			if err != nil {
				if tt.expectedErr == nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(tt.expectedErr.Error(), err.Error()) {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.expectedErr.Error(), err.Error(), tt.name)
				}
				return
			}
			if !strings.EqualFold(tt.expectedValue, *fakeSubnet.SubnetPropertiesFormat.NetworkSecurityGroup.ID) {
				t.Errorf("Expected NSG ID %s, got %s when processing %s testcase", tt.expectedValue, *fakeSubnet.SubnetPropertiesFormat.NetworkSecurityGroup.ID, tt.name)
			}
		})
	}
}

func TestGetSubnets(t *testing.T) {
	r := Reconciler{log: utillog.GetLogger()}
	for _, tt := range []struct {
		name             string
		machinelabel     string
		expectedMap      map[subnetDescriptor]bool
		expectedMasterRG string
		modify           func(*machinev1beta1.Machine, *machinev1beta1.Machine)
		expectedErr      error
	}{
		{
			name: "main path",
			expectedMap: map[subnetDescriptor]bool{
				{
					resourceGroup: "netRG",
					vnetName:      "workerVnet",
					subnetName:    "workerSubnet",
				}: true,
				{
					resourceGroup: "netRG",
					vnetName:      "masterVnet",
					subnetName:    "masterSubnet",
				}: false,
			},
			expectedMasterRG: "masterRG",
			modify:           func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {},
		},
		{
			name:             "missing providerSpec",
			expectedMap:      nil,
			expectedMasterRG: "",
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
				master.Spec.ProviderSpec.Value.Raw = []byte("")
			},
			expectedErr: fmt.Errorf("unexpected end of JSON input"),
		},
		{
			name:             "missing master nodes",
			expectedMap:      nil,
			expectedMasterRG: "masterRG",
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
				master.Labels = map[string]string{}
			},
			expectedErr: fmt.Errorf("master resource group not found"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			masterMachine := machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-0",
					Namespace: "openshift-machine-api",
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte("{\"resourceGroup\":\"masterRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"netRG\",\"vnet\":\"masterVnet\",\"subnet\":\"masterSubnet\"}"),
						},
					},
				},
			}
			workerMachine := machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-0",
					Namespace: "openshift-machine-api",
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte("{\"resourceGroup\":\"workerRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"netRG\",\"vnet\":\"workerVnet\",\"subnet\":\"workerSubnet\"}"),
						},
					},
				},
			}
			tt.modify(&workerMachine, &masterMachine)
			r.maocli = maofake.NewSimpleClientset(&workerMachine, &masterMachine)
			subnetMap, masterRG, err := r.getSubnets(context.Background())
			if err != nil {
				if tt.expectedErr == nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.expectedErr.Error(), err.Error(), tt.name)
				}
				return
			}
			if !strings.EqualFold(tt.expectedMasterRG, masterRG) {
				t.Errorf("Expected Master Resource Group %s, got %s when processing %s testcase", tt.expectedMasterRG, masterRG, tt.name)
			}
			if tt.expectedMap != nil {
				if len(tt.expectedMap) != len(subnetMap) {
					t.Errorf("Expected Map length %d, doesn't match result map length %d when processing %s testcase", len(tt.expectedMap), len(subnetMap), tt.name)
				}
				for subnet := range tt.expectedMap {
					value, present := subnetMap[subnet]
					if !present {
						t.Errorf("Subnet %s, %s, %s expected but not present in result when processing %s testcase", subnet.resourceGroup, subnet.vnetName, subnet.subnetName, tt.name)
					}
					if tt.expectedMap[subnet] != value {
						t.Errorf("Value of isMaster boolean doesn't match for subnet %s, %s, %s when processing %s testcase", subnet.resourceGroup, subnet.vnetName, subnet.subnetName, tt.name)
					}
				}
			}
		})
	}
}
