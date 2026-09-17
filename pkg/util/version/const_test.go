package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestOTelImage(t *testing.T) {
	const image = "/oss/otel/opentelemetry-collector-contrib@sha256:2597e3942ecda57062eed0171d2faf125ef6373772660cab5a175e185b10ba92"

	for _, domain := range []string{"", "arointsvc.azurecr.io", "arointsvc.azurecr.us"} {
		t.Run(domain, func(t *testing.T) {
			if got, want := OTelImage(domain), domain+image; got != want {
				t.Fatalf("OTelImage(%q) = %q, want %q", domain, got, want)
			}
		})
	}
}
