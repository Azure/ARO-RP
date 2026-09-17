package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"os"
	"os/exec"
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
		t.Fatal("generated VMSS does not contain the registry-relative MISE OTEL image")
	}

	var assignment string
	for _, line := range strings.Split(scriptRpVMSS, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "local -r otelimage=") {
			assignment = line
			break
		}
	}
	if assignment == "" {
		t.Fatal("MISE OTEL image assignment missing from the embedded startup script")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("startup substitution tests require bash")
	}
	for _, domain := range []string{"arointsvc.azurecr.io", "arointsvc.azurecr.us"} {
		for _, sourceDomain := range []string{"", "mcr.microsoft.com"} {
			t.Run(domain+"/"+sourceDomain, func(t *testing.T) {
				cmd := exec.Command(bash, "-c", "resolve_image() {\n"+assignment+"\nprintf '%s' \"$otelimage\"\n}\nresolve_image")
				cmd.Env = append(os.Environ(),
					"RPIMAGE="+domain+"/aro:test",
					"OTELIMAGE="+version.OTelImage(sourceDomain))
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("startup substitution failed: %v: %s", err, output)
				}
				if got, want := string(output), version.OTelImage(domain); got != want {
					t.Fatalf("startup image = %q, want %q", got, want)
				}
			})
		}
	}
}
