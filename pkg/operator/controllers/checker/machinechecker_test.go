package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"

	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMachineValid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		machine  *machinev1beta1.Machine
		wantErrs []error
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
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid VM size 'Standard_D2s_V3'"),
			},
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
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid disk size '64'"),
			},
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
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid image '{xyzcorp bananas   }'"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MachineChecker{}

			isMaster, err := isMasterRole(tt.machine)
			if err != nil {
				t.Fatal(err)
			}

			errs := r.machineValid(ctx, tt.machine, isMaster)

			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Errorf("MachineChecker.machineValid() = %v, want %v", errs, tt.wantErrs)
			}
		})
	}
}
