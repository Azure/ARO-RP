package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
			got := test.oc.UsesWorkloadIdentity()
			if got != test.want {
				t.Error(fmt.Errorf("got != want: %v != %v", got, test.want))
			}
		})
	}
}

func TestClusterMsiResourceId(t *testing.T) {
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name    string
		oc      *OpenShiftCluster
		wantErr string
	}{
		{
			name:    "error - cluster doc has nil Identity",
			oc:      &OpenShiftCluster{},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil Identity but nil Identity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &Identity{},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil Identity but empty Identity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &Identity{
					UserAssignedIdentities: UserAssignedIdentities{},
				},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			oc: &OpenShiftCluster{
				Identity: &Identity{
					UserAssignedIdentities: UserAssignedIdentities{
						"Hi hello I'm not a valid resource ID": ClusterUserAssignedIdentity{},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'Hi hello I'm not a valid resource ID' must start with '/'",
		},
		{
			name: "success",
			oc: &OpenShiftCluster{
				Identity: &Identity{
					UserAssignedIdentities: UserAssignedIdentities{
						miResourceId: ClusterUserAssignedIdentity{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_, err := tt.oc.ClusterMsiResourceId()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
