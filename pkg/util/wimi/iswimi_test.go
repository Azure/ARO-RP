package wimi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestIswimi(t *testing.T) {
	tests := []*struct {
		name string
		oc   api.OpenShiftCluster
		want bool
	}{
		{
			name: "Cluster is Workload Identity",
			oc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         nil,
				},
			},
			want: true,
		},
		{
			name: "Cluster is Service Principal",
			oc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := IsWimi(&test.oc)
			if got != test.want {
				t.Error(fmt.Errorf("got != want: %v != %v", got, test.want))
			}
		})
	}
}
