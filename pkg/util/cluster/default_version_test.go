package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestShouldInsertDefaultVersionInCosmosdb(t *testing.T) {
	tests := []struct {
		name         string
		versionsInDB []*api.OpenShiftVersion
		want         bool
	}{
		{
			name: "skip insert when another default already exists",
			versionsInDB: []*api.OpenShiftVersion{
				{
					Properties: api.OpenShiftVersionProperties{
						Version: "4.19.15",
						Default: true,
					},
				},
			},
			want: false,
		},
		{
			name: "repair local fallback version when it exists without default",
			versionsInDB: []*api.OpenShiftVersion{
				{
					Properties: api.OpenShiftVersionProperties{
						Version: version.DefaultInstallStream.Version.String(),
					},
				},
			},
			want: true,
		},
		{
			name: "insert when no default and no local fallback version exist",
			versionsInDB: []*api.OpenShiftVersion{
				{
					Properties: api.OpenShiftVersionProperties{
						Version: "4.16.30",
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInsertDefaultVersionInCosmosdb(tt.versionsInDB)
			if got != tt.want {
				t.Fatalf("shouldInsertDefaultVersionInCosmosdb() = %t, want %t", got, tt.want)
			}
		})
	}
}
