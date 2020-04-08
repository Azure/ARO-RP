package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestAPIVersionForType(t *testing.T) {
	tests := []struct {
		typ     string
		want    string
		wantErr string
	}{
		{
			typ:  "Microsoft.Network/dnsZones",
			want: APIVersions["Microsoft.Network/dnsZones"],
		},
		{
			typ:  "Microsoft.Network/loadBalancers",
			want: APIVersions["Microsoft.Network"],
		},
		{
			typ:     "Microsoft.Random/resources",
			wantErr: "API version not found for type Microsoft.Random/resources",
		},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got, err := APIVersionForType(tt.typ)
			if got != tt.want {
				t.Error(got)
			}
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
