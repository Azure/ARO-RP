package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/go-test/deep"
)

func TestInstallConfigMap(t *testing.T) {
	var expected = map[string]string{"install-config.yaml": "apiVersion: v1\nplatform:\n  azure:\n    region: \"testLocation\"\n"}

	r := installConfigCM("testNamespace", "testLocation")

	for _, err := range deep.Equal(r.StringData, expected) {
		t.Error(err)
	}
}
