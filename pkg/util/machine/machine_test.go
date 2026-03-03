package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

func TestIsMasterRole(t *testing.T) {
	type args struct {
		m *machinev1beta1.Machine
	}
	tests := []struct {
		name    string
		role    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "machine has worker role",
			role: "worker",
			args: args{
				m: workerMachine(),
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "machine has master role",
			role: "master",
			args: args{
				m: masterMachine(),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "machine has no role",
			role: "master",
			args: args{
				m: machineMissingRoleLabel(),
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasMasterRole(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasMasterRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasMasterRole() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func masterMachine() *machinev1beta1.Machine {
	return GetMachine("woo-hoo-master-2", true, true)
}

func workerMachine() *machinev1beta1.Machine {
	return GetMachine("foo-hero-worker-0", false, true)
}

func machineMissingRoleLabel() *machinev1beta1.Machine {
	return GetMachine("labelless-machine-boo-hoo", true, false)
}

func GetMachine(name string, isMaster bool, hasRole bool) *machinev1beta1.Machine {
	labels := map[string]string{}
	if hasRole {
		labels = map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"}
		if isMaster {
			labels = map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"}
		}
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "openshift-machine-api",
			Labels:    labels,
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{
					Raw: []byte(`{
"apiVersion": "machine.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
}`),
				},
			},
		},
	}
}
