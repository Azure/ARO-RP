package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func clusterVersion(version string) configv1.ClusterVersion {
	return configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: version,
				},
			},
		},
	}
}

func TestPodSecurityAdmissionControl(t *testing.T) {
	tests := []struct {
		name            string
		cv              configv1.ClusterVersion
		wantPodSecurity bool
		wantErr         string
	}{
		{
			name:            "cluster < 4.11, don't use pod security",
			cv:              clusterVersion("4.10.99"),
			wantPodSecurity: false,
		},
		{
			name:            "cluster >= 4.11, use pod security",
			cv:              clusterVersion("4.11.0"),
			wantPodSecurity: true,
		},
		{
			name:    "cluster version doesn't exist",
			cv:      configv1.ClusterVersion{},
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
		{
			name:    "invalid version",
			cv:      clusterVersion("abcd"),
			wantErr: `could not parse version "abcd"`,
		},
	}
	for _, tt := range tests {
		ctx := context.Background()
		client := ctrlfake.NewClientBuilder().WithObjects(&tt.cv).Build()

		gotUsePodSecurity, err := ShouldUsePodSecurityStandard(ctx, client)
		utilerror.AssertErrorMessage(t, err, tt.wantErr)

		if gotUsePodSecurity != tt.wantPodSecurity {
			t.Errorf("got: %v\nwanted:%v\n", gotUsePodSecurity, tt.wantPodSecurity)
		}
	}
}

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
