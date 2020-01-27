package encrypt

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	cipher, err := New(key)
	if err != nil {
		t.Error(err)
	}

	test := []byte("secert")
	encrypted, err := cipher.Encrypt(test)
	if err != nil {
		t.Error(err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Error(err)
	}

	if r := bytes.Compare(test, decrypted); r != 0 {
		t.Error("encryption roundTrip failed")
	}
}
