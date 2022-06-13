package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestUserPasswordFromB64(t *testing.T) {
	for _, tt := range []struct {
		name               string
		input              string
		wantUserOutput     string
		wantPasswordOutput string
		wantErr            error
	}{
		{
			name:               "ok",
			input:              base64.StdEncoding.EncodeToString([]byte("user:password")),
			wantUserOutput:     "user",
			wantPasswordOutput: "password",
		},
		{
			name:    "no password",
			input:   base64.StdEncoding.EncodeToString([]byte("user:")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "no password 2",
			input:   base64.StdEncoding.EncodeToString([]byte("nocolon")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "no user",
			input:   base64.StdEncoding.EncodeToString([]byte(":password")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "not valid b64",
			input:   "potato()+)(*",
			wantErr: fmt.Errorf("illegal base64 data at input byte 6"),
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			user, password, err := userPasswordFromB64(tt.input)
			if user != tt.wantUserOutput {
				t.Error(tt.name)
			}
			if password != tt.wantPasswordOutput {
				t.Error(tt.name)
			}
			if err != nil && tt.wantErr == nil {
				t.Error(tt.name)
			} else if err != nil && err.Error() != tt.wantErr.Error() {
				t.Error(tt.name)
			}
		})
	}
}

func TestExtractFromHeader(t *testing.T) {
	for _, tt := range []struct {
		name            string
		input           string
		wantBearerRealm string
		wantService     string
		wantErr         error
	}{
		{
			name:            "ok",
			input:           `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",service="docker-registry"`,
			wantBearerRealm: "https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",
			wantService:     "docker-registry",
		},
		{
			name:    "malformed header",
			input:   `="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",service="docker-r egistry"`,
			wantErr: fmt.Errorf("header is missing data"),
		},
		{
			name:    "missing service",
			input:   `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth"`,
			wantErr: fmt.Errorf("header is missing data"),
		},
		{
			name:    "missing service 2",
			input:   `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",`,
			wantErr: fmt.Errorf("header is missing data"),
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			realm, service, err := extractValuesFromAuthHeader(tt.input)
			if realm != tt.wantBearerRealm {
				t.Error(tt.name)
			}
			if service != tt.wantService {
				t.Error(tt.name)
			}
			if err != nil && tt.wantErr == nil {
				t.Error(tt.name)
			} else if err != nil && err.Error() != tt.wantErr.Error() {
				t.Error(tt.name)
			}
		})
	}
}
