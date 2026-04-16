package cluster

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
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
			name: "skip insert when local default version already exists",
			versionsInDB: []*api.OpenShiftVersion{
				{
					Properties: api.OpenShiftVersionProperties{
						Version: "4.17.44",
					},
				},
			},
			want: false,
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
