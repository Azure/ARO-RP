package pem

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"

	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestEncode(t *testing.T) {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	keyOut, err := Encode(validCaKey)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(keyOut, []byte("BEGIN RSA PRIVATE KEY")) {
		t.Fatal(string(keyOut))
	}

	certsOut, err := Encode(validCaCerts...)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(certsOut, []byte("BEGIN CERTIFICATE")) {
		t.Fatal(string(certsOut))
	}

	multiOut, err := Encode(validCaCerts[0], validCaCerts[0])
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Count(multiOut, []byte("BEGIN CERTIFICATE")) != 2 {
		t.Fatal(string(multiOut))
	}
}
