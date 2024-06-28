package identity

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestIsManagedIdentityCluster(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *api.OpenShiftCluster
		expected bool
	}{
		{
			name: "Managed Identity Cluster",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         nil,
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			expected: true,
		},
		{
			name: "Non-Managed Identity Cluster",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
					PlatformWorkloadIdentityProfile: nil,
				},
			},
			expected: false,
		},
		{
			name: "Nil Properties",
			cluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsManagedIdentityCluster(tt.cluster)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
