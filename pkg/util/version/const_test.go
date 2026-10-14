package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"
)

func TestOTelImage(t *testing.T) {
	image := OTelImage("")
	if !strings.HasPrefix(image, "/") {
		t.Fatalf("OTelImage(\"\") = %q, want a registry-relative image reference", image)
	}

	for _, domain := range []string{"arointsvc.azurecr.io", "arointsvc.azurecr.us"} {
		t.Run(domain, func(t *testing.T) {
			if got, want := OTelImage(domain), domain+image; got != want {
				t.Fatalf("OTelImage(%q) = %q, want %q", domain, got, want)
			}
		})
	}
}
