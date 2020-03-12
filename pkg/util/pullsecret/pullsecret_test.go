package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestSetRegistryAuth(t *testing.T) {
	original := `{
	"auths": {
	  "arosvc.azurecr.io": {"auth": "x"},
	  "registry.connect.redhat.com": {"auth": "y"},
	  "registry.redhat.io": {"auth": "z"}
	}
  }
  `

	tests := []struct {
		name     string
		registry string
		username string
		password string
		want     string
		wantErr  bool
	}{
		{
			name:     "replace",
			registry: "arosvc.azurecr.io",
			username: "fred",
			password: "enter",
			want:     `"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}`,
		},
		{
			name:     "add",
			registry: "arosvc-int.azurecr.io",
			username: "fred",
			password: "enter",
			want:     `"arosvc-int.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetRegistryAuth(original, &api.RegistryProfile{Name: tt.registry, Username: tt.username, Password: api.SecureString(tt.password)})
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRegistryAuth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("SetRegistryAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}
