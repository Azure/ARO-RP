package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestEnvironmentType(t *testing.T) {
	p := &prod{envType: EnvironmentTypeProduction}
	out := p.Type()

	matching := EnvironmentTypeDevelopment | EnvironmentTypeProduction
	if out&matching == 0 {
		t.Fatal("Doesn't match expected mask")
	}

	matching = EnvironmentTypeIntegration | EnvironmentTypeDevelopment
	if out&matching != 0 {
		t.Fatal("Doesn't match expected mask")
	}
}
