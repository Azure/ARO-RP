package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestOpenShiftVersion(t *testing.T) {
	_, err := ParseVersion(OpenShiftVersion)
	if err != nil {
		t.Error(err)
	}
}
