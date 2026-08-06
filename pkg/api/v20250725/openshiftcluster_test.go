package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

func TestIsWorkloadIdentity(t *testing.T) {
	tests := []*struct {
		name string
		oc   OpenShiftCluster
		want bool
	}{
		{
			name: "Cluster is Workload Identity",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         nil,
				}},
			},
			want: true,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         &generated.ServicePrincipalProfile{},
				}},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         nil,
				}},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         &generated.ServicePrincipalProfile{},
				}},
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
