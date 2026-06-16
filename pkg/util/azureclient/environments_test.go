package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNewTokenCredentialRequiresAzureTokenCredentials(t *testing.T) {
	// NewTokenCredential sets RequireAzureTokenCredentials, so NewDefaultAzureCredential
	// errors when AZURE_TOKEN_CREDENTIALS is unset.
	if v, ok := os.LookupEnv("AZURE_TOKEN_CREDENTIALS"); ok {
		os.Unsetenv("AZURE_TOKEN_CREDENTIALS")
		defer os.Setenv("AZURE_TOKEN_CREDENTIALS", v)
	}

	for _, env := range []*AROEnvironment{&PublicCloud, &USGovernmentCloud} {
		t.Run(env.Name, func(t *testing.T) {
			_, err := env.NewTokenCredential()
			if err == nil {
				t.Errorf("NewTokenCredential() for %s should fail when AZURE_TOKEN_CREDENTIALS is unset, indicating RequireAzureTokenCredentials is set", env.Name)
			} else if !strings.Contains(err.Error(), "AZURE_TOKEN_CREDENTIALS") {
				t.Errorf("NewTokenCredential() for %s returned unexpected error: %v", env.Name, err)
			}
		})
	}
}

func TestEnvironmentFromName(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr string
		azEnv   string
	}{
		{
			name:    "fail: invalid az environment",
			azEnv:   "NEVERLAND",
			wantErr: `cloud environment "NEVERLAND" is unsupported by ARO`,
		},
		{
			name:  "pass: public cloud az environment",
			azEnv: azure.PublicCloud.Name,
		},
		{
			name:  "pass: US government cloud",
			azEnv: azure.USGovernmentCloud.Name,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EnvironmentFromName(tt.azEnv)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
