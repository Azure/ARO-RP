package oidcbuilder

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"go.uber.org/mock/gomock"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestKeyIDFromPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	keyID, err := keyIDFromPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	invalidPublicKey := rsa.PublicKey{}

	for _, tt := range []struct {
		name      string
		publicKey interface{}
		wantkid   string
		wantErr   string
	}{
		{
			name:      "Success",
			publicKey: &privateKey.PublicKey,
			wantkid:   keyID,
		},
		{
			name:      "Failed to serialize public key",
			publicKey: &invalidPublicKey,
			wantErr:   "Failed to serialize public key to DER format: asn1: structure error: empty integer",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			kid, err := keyIDFromPublicKey(tt.publicKey)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantkid != kid {
				t.Fatalf("Expected KeyID and returned KeyID doesn't match")
			}
		})
	}
}
