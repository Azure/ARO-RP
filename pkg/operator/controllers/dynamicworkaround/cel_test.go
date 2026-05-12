package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

// mustVersion parses v or fails the test. Predicate tests need real Version
// instances since ClusterFacts.ClusterVersion is consumed as a version.Version.
func mustVersion(t *testing.T, v string) version.Version {
	t.Helper()
	parsed, err := version.ParseVersion(v)
	if err != nil {
		t.Fatalf("ParseVersion(%q): %v", v, err)
	}
	return parsed
}

func TestCompileCEL(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr string
	}{
		{name: "happy path: bool literal", expr: "true"},
		{name: "happy path: equality", expr: `ipsecMode == "Full"`},
		{name: "happy path: combination", expr: `region == "eastus" && architectureVersion == 2`},
		{name: "happy path: versionAtLeast", expr: `versionAtLeast(clusterVersion, "4.16.0")`},
		{name: "happy path: versionLessThan", expr: `versionLessThan(clusterVersion, "4.18.0")`},
		{name: "happy path: range", expr: `versionAtLeast(clusterVersion, "4.16.0") && versionLessThan(clusterVersion, "4.18.0")`},

		{name: "syntax error", expr: `region ===`, wantErr: "compile CEL"},
		{name: "unknown variable", expr: `nonsense == "x"`, wantErr: "compile CEL"},
		{name: "wrong return type: string", expr: `region`, wantErr: "must return bool"},
		{name: "wrong return type: int", expr: `architectureVersion`, wantErr: "must return bool"},
		{name: "unknown function", expr: `frobnicate(region)`, wantErr: "compile CEL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileCEL(tt.expr)
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

// TestEvalCEL exercises evalCEL (compile + run) for every shape of expression
// catalog authors are expected to write. It is the workhorse test for the CEL
// surface; production reconcile goes through Predicates.Eval which uses the
// same compileCEL + runCELProgram pair under the hood.
func TestEvalCEL(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		facts   ClusterFacts
		want    bool
		wantErr string
	}{
		{name: "trivial true", expr: "true", facts: ClusterFacts{}, want: true},
		{name: "trivial false", expr: "false", facts: ClusterFacts{}, want: false},
		{
			name:  "region equality matches",
			expr:  `region == "eastus"`,
			facts: ClusterFacts{Location: "eastus"},
			want:  true,
		},
		{
			name:  "region equality mismatches",
			expr:  `region == "eastus"`,
			facts: ClusterFacts{Location: "westus"},
			want:  false,
		},
		{
			name:  "architecture predicate",
			expr:  `architectureVersion == 2`,
			facts: ClusterFacts{ArchitectureVersion: 2},
			want:  true,
		},
		{
			name:  "ipsec absent maps to empty string",
			expr:  `ipsecMode == ""`,
			facts: ClusterFacts{IPSecMode: ""},
			want:  true,
		},
		{
			name:  "ipsec literal match",
			expr:  `ipsecMode == "Full"`,
			facts: ClusterFacts{IPSecMode: "Full"},
			want:  true,
		},
		{
			name:  "negated equality",
			expr:  `ipsecMode != "Disabled"`,
			facts: ClusterFacts{IPSecMode: "Full"},
			want:  true,
		},

		// versionAtLeast / versionLessThan
		{
			name:  "versionAtLeast: cluster newer",
			expr:  `versionAtLeast(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.17.0")},
			want:  true,
		},
		{
			name:  "versionAtLeast: cluster older",
			expr:  `versionAtLeast(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.15.99")},
			want:  false,
		},
		{
			name:  "versionAtLeast: exactly equal",
			expr:  `versionAtLeast(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.16.0")},
			want:  true,
		},
		{
			name:  "versionLessThan: strict",
			expr:  `versionLessThan(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.15.99")},
			want:  true,
		},
		{
			name:  "versionLessThan: at boundary",
			expr:  `versionLessThan(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.16.0")},
			want:  false,
		},
		{
			// Crucial fail-closed property: unknown clusterVersion ("" in CEL
			// input) must NOT silently match a versionAtLeast call.
			name:  "versionAtLeast: unknown cluster version fails closed",
			expr:  `versionAtLeast(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: nil},
			want:  false,
		},
		{
			name:  "versionLessThan: unknown cluster version fails closed",
			expr:  `versionLessThan(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{ClusterVersion: nil},
			want:  false,
		},
		{
			// 4.10.0 vs 4.9.0 is the textbook case where lexicographic
			// string comparison gets the wrong answer; the helper must use
			// semver ordering.
			name:  "versionAtLeast: semver ordering not lex",
			expr:  `versionAtLeast(clusterVersion, "4.9.0")`,
			facts: ClusterFacts{ClusterVersion: mustVersion(t, "4.10.0")},
			want:  true,
		},

		// Combined expressions
		{
			name: "combined expression: all must match",
			expr: `region == "eastus" && architectureVersion == 2 && versionAtLeast(clusterVersion, "4.16.0")`,
			facts: ClusterFacts{
				ClusterVersion:      mustVersion(t, "4.17.0"),
				Location:            "eastus",
				ArchitectureVersion: 2,
			},
			want: true,
		},
		{
			name: "combined expression: short-circuits on first false",
			expr: `region == "westus" && architectureVersion == 2`,
			facts: ClusterFacts{
				Location:            "eastus",
				ArchitectureVersion: 2,
			},
			want: false,
		},

		// Errors at eval time
		{
			name:    "versionAtLeast: bad target",
			expr:    `versionAtLeast(clusterVersion, "not-a-version")`,
			facts:   ClusterFacts{ClusterVersion: mustVersion(t, "4.17.0")},
			wantErr: "versionAtLeast",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalCEL(context.Background(), tt.expr, tt.facts)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
			if tt.wantErr != "" {
				return
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
