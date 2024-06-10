package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"
)

func TestToPtr(t *testing.T) {
	input := []byte("Test String")
	output := ToPtr(input)

	if output == &input {
		t.Errorf("Value returned by ToPtr does not matches the expected pointer value")
	}

	if !bytes.Equal(input, *output) {
		t.Errorf("Input bytes doesn't match with the bytes value for the pointer returned by ToPtr")
	}
}
