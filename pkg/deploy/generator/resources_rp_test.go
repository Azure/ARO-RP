package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestRPVMSSOTelImage(t *testing.T) {
	resource := (&generator{production: true}).rpVMSS()
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "'OTELIMAGE=''"+version.OTelImage("")+"''") {
		t.Fatal("generated VMSS does not contain the configured MISE OTEL image reference")
	}
}
