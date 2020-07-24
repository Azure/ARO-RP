package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMachineValid(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name      string
		machine   *machinev1beta1.Machine
		wantErr   bool
		wantValid bool
		wantMsgs  []string
	}{
		{
			name: "valid",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
					Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
			wantValid: true,
			wantMsgs:  []string{},
		},
		{
			name: "wrong vmSize",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D2s_V3"
}`),
						},
					},
				},
			},
			wantValid: false,
			wantMsgs:  []string{"the machine foo-hx8z7-master-0 VM size 'Standard_D2s_V3' is invalid"},
		},
		{
			name: "wrong diskSize",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 64
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
			wantValid: false,
			wantMsgs:  []string{"the machine foo-hx8z7-master-0 disk size '64' is invalid"},
		},
		{
			name: "wrong image publisher",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 128
},
"image": {
"publisher": "xyzcorp",
"offer": "bananas"
},
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
			wantValid: false,
			wantMsgs:  []string{"the machine foo-hx8z7-master-0 image '{xyzcorp bananas   }' is invalid"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MachineChecker{
				developmentMode: false,
			}
			gotSup, gotMsg, err := r.machineValid(ctx, tt.machine)
			if (err != nil) != tt.wantErr {
				t.Errorf("MachineChecker.machineValid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSup != tt.wantValid {
				t.Errorf("MachineChecker.machineValid() = %v, want %v", gotSup, tt.wantValid)
			}
			if !reflect.DeepEqual(gotMsg, tt.wantMsgs) {
				t.Errorf("MachineChecker.machineValid() = %v, want %v", gotMsg, tt.wantMsgs)
			}
		})
	}
}
