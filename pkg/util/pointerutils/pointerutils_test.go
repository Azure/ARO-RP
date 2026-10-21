package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"
)

func TestToSlicePtr(t *testing.T) {
	input := []byte("Test String")
	output := ToSlicePtr([][]byte{input})

	if !bytes.Equal(input, *output[0]) {
		t.Errorf("Input bytes doesn't match with the bytes value for the pointer returned by ToSlicePtr")
	}
}
