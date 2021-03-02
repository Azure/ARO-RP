package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
)

func TestLastTokenByte(t *testing.T) {
	result := LastTokenByte("a/b/c/d", '/')
	want := "d"
	if result != want {
		t.Errorf("want %s, got %s", want, result)
	}
}

func TestUniqueSlice(t *testing.T) {
	result := UniqueSlice([]string{"foo", "foo", "bar"})
	want := []string{"foo", "bar"}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("want %s, got %s", want, result)
	}
}
