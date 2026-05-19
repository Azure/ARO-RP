package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	apitesterror "github.com/Azure/ARO-RP/pkg/api/test/error"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
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
			name:    "error - cluster doc has nil ManagedServiceIdentity",
			oc:      &OpenShiftCluster{},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil ManagedServiceIdentity but nil ManagedServiceIdentity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil ManagedServiceIdentity but empty ManagedServiceIdentity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{},
				},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil Identity but two MSIs in Identity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						miResourceId: {
							ClientID:    "",
							PrincipalID: "",
						},
						"secondEntry": {
							ClientID:    "",
							PrincipalID: "",
						},
					},
				},
			},
			wantErr: "unexpectedly found more than one cluster MSI in cluster doc",
		},
		{
			name: "success",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						miResourceId: {
							ClientID:    "",
							PrincipalID: "",
						},
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
			apitesterror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestHasUserAssignedIdentities(t *testing.T) {
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name       string
		oc         *OpenShiftCluster
		wantResult bool
	}{
		{
			name:       "false - cluster doc has nil Identity",
			oc:         &OpenShiftCluster{},
			wantResult: false,
		},
		{
			name: "false - cluster doc has non-nil Identity but nil Identity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{},
			},
			wantResult: false,
		},
		{
			name: "false - cluster doc has non-nil Identity but empty Identity.UserAssignedIdentities",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{},
				},
			},
			wantResult: false,
		},
		{
			name: "true",
			oc: &OpenShiftCluster{
				Identity: &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						miResourceId: {},
					},
				},
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			got := tt.oc.HasUserAssignedIdentities()
			if got != tt.wantResult {
				t.Error(fmt.Errorf("got != want: %v != %v", got, tt.wantResult))
			}
		})
	}
}

func TestPutRegistryProfile(t *testing.T) {
	a := require.New(t)

	newProfile := &RegistryProfile{
		Name:      "arointsvc.example.com",
		Username:  "token-22222222-2222-2222-2222-222222220001",
		IssueDate: pointerutils.ToPtr(time.Unix(1, 0)),
	}

	ocWithProfile := &OpenShiftCluster{
		Properties: OpenShiftClusterProperties{
			RegistryProfiles: []*RegistryProfile{
				{
					Name:     "arointsvc.example.com",
					Username: "foo",
				},
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}
	ocWithoutProfile := &OpenShiftCluster{
		Properties: OpenShiftClusterProperties{
			RegistryProfiles: []*RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}

	// If it doesn't exist, it appends it
	ocWithoutProfile.PutRegistryProfile(newProfile)
	a.Len(ocWithoutProfile.Properties.RegistryProfiles, 2)

	// If it does exist, it replaces it
	ocWithProfile.PutRegistryProfile(newProfile)
	a.Len(ocWithProfile.Properties.RegistryProfiles, 2)

	// Check that it has been replaced
	aLongTimeAgo := time.UnixMilli(1000)

	a.Equal(ocWithProfile.Properties.RegistryProfiles,
		[]*RegistryProfile{
			{
				Name:      "arointsvc.example.com",
				Username:  "token-22222222-2222-2222-2222-222222220001",
				IssueDate: &aLongTimeAgo,
			},
			{
				Name:     "notwanted.example.com",
				Username: "other",
			},
		})
}

func TestGetRegistryProfiles(t *testing.T) {
	a := require.New(t)

	ocWithProfile := &OpenShiftCluster{
		Properties: OpenShiftClusterProperties{
			RegistryProfiles: []*RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
				{
					Name:     "arointsvc.example.com",
					Username: "foo",
				},
			},
		},
	}
	ocWithoutProfile := &OpenShiftCluster{
		Properties: OpenShiftClusterProperties{
			RegistryProfiles: []*RegistryProfile{
				{
					Name:     "notwanted.example.com",
					Username: "other",
				},
			},
		},
	}

	// GetRegistryProfile finds it successfully
	r := ocWithProfile.GetRegistryProfile("arointsvc.example.com")
	a.NotNil(r)
	a.Equal("arointsvc.example.com", r.Name)
	a.Equal("foo", r.Username)

	// GetRegistryProfile can't find it as it doesn't exist
	r = ocWithoutProfile.GetRegistryProfile("arointsvc.example.com")
	a.Nil(r)
}
