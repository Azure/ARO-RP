package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"
	"testing"
)

// validWorkaround is a helper that returns a structurally-valid Workaround so
// each test case only has to specify the field it's exercising.
func validWorkaround(name string) Workaround {
	return Workaround{
		Name:              name,
		MachineConfigName: "99-" + name,
		Role:              "worker",
		Ignition:          json.RawMessage(`{"ignition":{"version":"3.2.0"}}`),
	}
}

func validCatalog() Catalog {
	return Catalog{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "test-1",
		Workarounds:    []Workaround{validWorkaround("test-wa")},
	}
}

func TestCatalogValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Catalog)
		wantErr string // substring; empty means "no error"
	}{
		{
			name:   "happy path",
			mutate: func(c *Catalog) {},
		},
		{
			name:   "empty workaround list is valid",
			mutate: func(c *Catalog) { c.Workarounds = nil },
		},
		{
			name:    "wrong schemaVersion",
			mutate:  func(c *Catalog) { c.SchemaVersion = "v2" },
			wantErr: "unsupported schemaVersion",
		},
		{
			name:    "missing catalogVersion",
			mutate:  func(c *Catalog) { c.CatalogVersion = "" },
			wantErr: "catalogVersion must be non-empty",
		},
		{
			name: "too many workarounds",
			mutate: func(c *Catalog) {
				c.Workarounds = make([]Workaround, MaxWorkarounds+1)
				for i := range c.Workarounds {
					c.Workarounds[i] = validWorkaround("wa")
				}
			},
			wantErr: "too many workarounds",
		},
		{
			name: "invalid workaround name",
			mutate: func(c *Catalog) {
				c.Workarounds[0].Name = "NOT_A_DNS_LABEL"
			},
			wantErr: "is not a valid DNS label",
		},
		{
			name: "duplicate workaround name",
			mutate: func(c *Catalog) {
				w := validWorkaround("dup")
				c.Workarounds = []Workaround{w, w}
			},
			wantErr: "duplicated",
		},
		{
			name: "invalid machineConfigName",
			mutate: func(c *Catalog) {
				c.Workarounds[0].MachineConfigName = "Bad_Name"
			},
			wantErr: "machineConfigName",
		},
		{
			name: "invalid role",
			mutate: func(c *Catalog) {
				c.Workarounds[0].Role = "infra"
			},
			wantErr: `role must be "master" or "worker"`,
		},
		{
			name: "empty ignition",
			mutate: func(c *Catalog) {
				c.Workarounds[0].Ignition = nil
			},
			wantErr: "ignition is required",
		},
		{
			name: "invalid ignition JSON",
			mutate: func(c *Catalog) {
				c.Workarounds[0].Ignition = json.RawMessage(`not-json`)
			},
			wantErr: "ignition is not valid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validCatalog()
			tt.mutate(&c)

			err := c.Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNameRegex(t *testing.T) {
	// Spot-check the regex; the goal here is to catch accidental future
	// loosening (e.g. someone adds underscores to allowlist) more than to
	// exhaustively verify DNS label semantics.
	good := []string{"a", "ab", "a-b", "99-aro-fix", "abcdefghij0123456789-x"}
	bad := []string{"", "-a", "a-", "A", "a_b", "a.b", "a b", "aro!"}

	for _, s := range good {
		if !nameRegex.MatchString(s) {
			t.Errorf("expected %q to be a valid name", s)
		}
	}
	for _, s := range bad {
		if nameRegex.MatchString(s) {
			t.Errorf("expected %q to be rejected", s)
		}
	}
}
