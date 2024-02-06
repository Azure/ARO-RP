package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func aroCluster(domains []string) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			GatewayDomains: domains,
		},
	}
}

func TestGatewayEnabled(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *arov1alpha1.Cluster
		wantEnabled bool
	}{
		{
			name:    "gateway disabled",
			cluster: aroCluster([]string{}),
		},
		{
			name:        "gateway enabled",
			cluster:     aroCluster([]string{"domain1", "domain2"}),
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		gotEnabled := GatewayEnabled(tt.cluster)
		if gotEnabled != tt.wantEnabled {
			t.Errorf("got: %v\nwant: %v\n", gotEnabled, tt.wantEnabled)
		}
	}
}
