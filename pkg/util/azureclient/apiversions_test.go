package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestAPIVersion(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{
			typ:  "Microsoft.Network/dnsZones",
			want: apiVersions["microsoft.network/dnszones"],
		},
		{
			typ:  "Microsoft.Network/loadBalancers",
			want: apiVersions["microsoft.network"],
		},
		{
			typ:  "Microsoft.Network/privateDnsZones/virtualNetworkLinks",
			want: apiVersions["microsoft.network/privatednszones"],
		},
		{
			typ:  "Microsoft.ContainerRegistry/registries/replications",
			want: apiVersions["microsoft.containerregistry"],
		},
		{
			typ:  "Microsoft.Compute/galleries",
			want: apiVersions["microsoft.compute/galleries"],
		},
		{
			typ: "Microsoft.Random/resources",
		},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := APIVersion(tt.typ)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}
