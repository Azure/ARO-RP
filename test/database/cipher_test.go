package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestFakeCipher(t *testing.T) {
	c := &fakeCipher{}

	encrypted, _ := c.Encrypt([]byte{'h', 'i'})
	if string(encrypted) != "FAKEhi" {
		t.Error(string(encrypted))
	}

	decrypted, _ := c.Decrypt(encrypted)
	if string(decrypted) != "hi" {
		t.Error(string(decrypted))
	}
}
