package identity

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestIsManagedWorkloadIdentityEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *api.OpenShiftCluster
		expected bool
	}{
		{
			name: "Workload Identity Enabled",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         nil,
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
				Identity: &api.Identity{},
			},
			expected: true,
		},
		{
			name: "Service Principal Profile not nil",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
					PlatformWorkloadIdentityProfile: nil,
				},
				Identity: nil,
			},
			expected: false,
		},
		{
			name: "PlatformWorkloadIdentityProfile is nil",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         nil,
					PlatformWorkloadIdentityProfile: nil,
				},
				Identity: &api.Identity{},
			},
			expected: false,
		},
		{
			name: "Identity is nil",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         nil,
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
				Identity: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsManagedWorkloadIdentityEnabled(tt.cluster)
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}
