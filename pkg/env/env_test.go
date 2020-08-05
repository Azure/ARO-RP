package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestEnvironmentType(t *testing.T) {
	p := &prod{envType: environmentTypeProduction}
	out := p.IsDevelopment()
	if out != false {
		t.Fatal("didn't return expected value")
	}

	p = &prod{envType: environmentTypeIntegration}
	out = p.IsDevelopment()
	if out != false {
		t.Fatal("didn't return expected value")
	}

	p = &prod{envType: environmentTypeDevelopment}
	out = p.IsDevelopment()
	if out != true {
		t.Fatal("didn't return expected value")
	}
}

func TestEnvironmentShouldDeployDenyAssignment(t *testing.T) {
	p := &prod{envType: environmentTypeProduction}
	out := p.ShouldDeployDenyAssignment()
	if out != true {
		t.Fatal("didn't return expected value")
	}

	p = &prod{envType: environmentTypeIntegration}
	out = p.ShouldDeployDenyAssignment()
	if out != false {
		t.Fatal("didn't return expected value")
	}

	p = &prod{envType: environmentTypeDevelopment}
	out = p.ShouldDeployDenyAssignment()
	if out != false {
		t.Fatal("didn't return expected value")
	}
}
