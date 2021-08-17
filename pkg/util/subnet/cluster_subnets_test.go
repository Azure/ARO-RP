package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	subscriptionId    = "0000000-0000-0000-0000-000000000000"
	vnetResourceGroup = "vnet-rg"
	vnetName          = "vnet"
	subnetNameWorker  = "worker"
	subnetNameMaster  = "master"
)

func TestListFromCluster(t *testing.T) {
	for _, tt := range []struct {
		name         string
		machinelabel string
		expectedMap  []Subnet
		modify       func(*machinev1beta1.Machine, *machinev1beta1.Machine)
		expectedErr  error
	}{
		{
			name: "main path",
			expectedMap: []Subnet{
				{
					ResourceID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker,
				},
				{
					ResourceID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster,
				},
			},
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {},
		},
		{
			name:        "missing providerSpec",
			expectedMap: nil,
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
				master.Spec.ProviderSpec.Value.Raw = []byte("")
			},
			expectedErr: fmt.Errorf("unexpected end of JSON input"),
		},
		{
			name:        "missing master nodes",
			expectedMap: nil,
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

			m := kubeManager{
				maocli: maofake.NewSimpleClientset(&workerMachine, &masterMachine),
			}

			subnets, err := m.ListFromCluster(context.Background())
			if err != nil {
				if tt.expectedErr == nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.expectedErr.Error(), err.Error(), tt.name)
				}
				return
			}

			if tt.expectedMap != nil {
				if len(tt.expectedMap) != len(subnets) {
					t.Errorf("Expected Map length %d, doesn't match result map length %d when processing %s testcase", len(tt.expectedMap), len(subnets), tt.name)
				}
				if cmp.Equal(tt.expectedErr, subnets) {
					t.Fatal(cmp.Diff(tt.expectedErr, subnets))
				}
			}
		})
	}
}
