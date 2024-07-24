package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	for _, tt := range []struct {
		name  string
		want  bool
		state ProvisioningState
	}{
		{
			name:  "Success is Terminal",
			want:  true,
			state: ProvisioningStateSucceeded,
		},
		{
			name:  "Failed is Terminal",
			want:  true,
			state: ProvisioningStateFailed,
		},
		{
			name:  "Creating is Non-Terminal",
			want:  false,
			state: ProvisioningStateCreating,
		},
		{
			name:  "Updating is Non-Terminal",
			want:  false,
			state: ProvisioningStateUpdating,
		},
		{
			name:  "AdminUpdating is Non-Terminal",
			want:  false,
			state: ProvisioningStateAdminUpdating,
		},
		{
			name:  "Deleting is Non-Terminal",
			want:  false,
			state: ProvisioningStateDeleting,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsTerminal() != tt.want {
				t.Fatalf("%s isTerminal wants != %t", tt.state, tt.want)
			}
		})
	}
}

func TestIsWorkloadIdentity(t *testing.T) {
	tests := []*struct {
		name string
		oc   OpenShiftCluster
		want bool
	}{
		{
			name: "Cluster is Workload Identity",
			oc: OpenShiftCluster{
				Properties: OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         nil,
				},
			},
			want: true,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				Properties: OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         &ServicePrincipalProfile{},
				},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				Properties: OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         nil,
				},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				Properties: OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         &ServicePrincipalProfile{},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.oc.IsWorkloadIdentity()
			if got != test.want {
				t.Error(fmt.Errorf("got != want: %v != %v", got, test.want))
			}
		})
	}
}
