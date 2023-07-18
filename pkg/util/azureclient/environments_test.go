package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

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
