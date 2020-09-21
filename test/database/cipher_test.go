package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestFakeCipher(t *testing.T) {
	c := &fakeCipher{}

	encrypted, _ := c.Encrypt([]byte{'f', 'o', 'o'})
	if string(encrypted) != "FAKEfoo" {
		t.Error(string(encrypted))
	}

	decrypted, _ := c.Decrypt(encrypted)
	if string(decrypted) != "foo" {
		t.Error(string(decrypted))
	}
}
