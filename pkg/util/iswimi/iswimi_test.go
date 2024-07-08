package iswimi

import (
	"fmt"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestIswimi(t *testing.T) {
	tests := []struct {
		name    string
		cluster api.OpenShiftClusterProperties
		want    bool
	}{
		{
			name: "Cluster is Workload Identity",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				ServicePrincipalProfile:         nil,
			},
			want: true,
		},
		{
			name: "Cluster is Classic",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: nil,
				ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := IsWimi(test.cluster)
			if got != test.want {
				t.Error(fmt.Errorf("got != want: %v != %v", got, test.want))
			}
		})
	}
}
