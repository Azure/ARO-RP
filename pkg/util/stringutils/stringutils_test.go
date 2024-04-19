package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestLastTokenByte(t *testing.T) {
	result := LastTokenByte("a/b/c/d", '/')
	want := "d"
	if result != want {
		t.Errorf("want %s, got %s", want, result)
	}
}

func TestIsResourceIDFormatted(t *testing.T) {
	result := IsResourceIDFormatted("/subscriptions/subscriptionID/resourceGroups/resourcegroupname/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourcename")
	want := true
	if result != want {
		t.Errorf("want %t, got %t\n", want, result)
	}

	result = IsResourceIDFormatted("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	want = false
	if result != want {
		t.Errorf("want %t, got %t\n", want, result)
	}
}
