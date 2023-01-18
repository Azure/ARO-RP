package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

const (
	subscriptionId    = "0000000-0000-0000-0000-000000000000"
	vnetResourceGroup = "vnet-rg"
	vnetName          = "vnet"
	subnetNameWorker  = "workerSubnet"
	subnetNameMaster  = "masterSubnet"
)

func TestListFromCluster(t *testing.T) {
	for _, tt := range []struct {
		name         string
		machinelabel string
		expect       []Subnet
		modify       func(*machinev1beta1.Machine, *machinev1beta1.Machine)
		wantErr      string
	}{
		{
			name: "main path",
			expect: []Subnet{
				{
					ResourceID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker,
				},
				{
					ResourceID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster,
					IsMaster:   true,
				},
			},
		},
		{
			name:   "master missing providerSpec",
			expect: nil,
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
				master.Spec.ProviderSpec.Value.Raw = []byte("")
			},
			wantErr: "json: error calling MarshalJSON for type *runtime.RawExtension: unexpected end of JSON input",
		},
		{
			name:   "worker missing providerSpec",
			expect: nil,
			modify: func(worker *machinev1beta1.Machine, master *machinev1beta1.Machine) {
				worker.Spec.ProviderSpec.Value.Raw = []byte("")
			},
			wantErr: "json: error calling MarshalJSON for type *runtime.RawExtension: unexpected end of JSON input",
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
						Value: &kruntime.RawExtension{
							Raw: []byte("{\"resourceGroup\":\"masterRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"vnet-rg\",\"vnet\":\"vnet\",\"subnet\":\"masterSubnet\"}"),
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
						Value: &kruntime.RawExtension{
							Raw: []byte("{\"resourceGroup\":\"workerRG\",\"publicIP\":false,\"osDisk\":{\"diskSizeGB\": 1024,\"managedDisk\":{\"storageAccountType\": \"Premium_LRS\"},\"osType\":\"Linux\"},\"image\":{\"offer\": \"aro4\",\"publisher\": \"azureopenshift\", \"resourceID\": \"\", \"sku\": \"aro_43\", \"version\": \"43.81.20200311\"},\"networkResourceGroup\":\"vnet-rg\",\"vnet\":\"vnet\",\"subnet\":\"workerSubnet\"}"),
						},
					},
				},
			}
			if tt.modify != nil {
				tt.modify(&workerMachine, &masterMachine)
			}

			m := kubeManager{
				client:         fake.NewClientBuilder().WithObjects(&workerMachine, &masterMachine).Build(),
				subscriptionID: subscriptionId,
			}

			subnets, err := m.List(context.Background())
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if !cmp.Equal(tt.expect, subnets) {
				t.Fatal(cmp.Diff(tt.expect, subnets))
			}
		})
	}
}
