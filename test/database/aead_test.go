package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestFakeAEAD(t *testing.T) {
	c := &fakeAEAD{}

	encrypted, _ := c.Seal([]byte{'f', 'o', 'o'})
	if string(encrypted) != "FAKEfoo" {
		t.Error(string(encrypted))
	}

	decrypted, _ := c.Open(encrypted)
	if string(decrypted) != "foo" {
		t.Error(string(decrypted))
	}
}
