package token

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/test/util/token"
)

func TestExtractClaims(t *testing.T) {
	dummyObjectId := "1234567890"
	validTestToken, err := token.CreateTestToken(dummyObjectId, nil)
	if err != nil {
		t.Errorf("Error creating test token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantOid    string
		wantClaims map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "Can extract oid from a valid token",
			token:      validTestToken,
			wantOid:    dummyObjectId,
			wantClaims: map[string]interface{}{"example_claim": "example_value"},
			wantErr:    false,
		},
		{
			name:       "Return an error when given an invalid jwt",
			token:      "invalid",
			wantOid:    "",
			wantClaims: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractClaims(tt.token)
			t.Log(got, err)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expect an error but got nothing")
				}
			} else {
				if got.ObjectId != tt.wantOid {
					t.Errorf("Got oid: %q, want %q", got.ObjectId, tt.wantOid)
				}
				if diff := cmp.Diff(got.ClaimNames, tt.wantClaims); diff != "" {
					t.Errorf("Got claimNames: %q, want %q", got.ClaimNames, tt.wantClaims)
				}
				if err != nil {
					t.Errorf("Expect no error but got one")
				}
			}
		})
	}
}
