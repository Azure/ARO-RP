package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"
)

func TestParsePredicates(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantLen int    // expected number of compiled predicates
		wantErr string // substring; empty means must succeed
	}{
		{
			name:    "empty value yields empty map",
			raw:     "",
			wantLen: 0,
		},
		{
			name:    "single entry",
			raw:     `{"ipsec-mtu-fix": "ipsecMode == \"Full\""}`,
			wantLen: 1,
		},
		{
			name:    "multiple entries",
			raw:     `{"a": "true", "b": "region == \"eastus\""}`,
			wantLen: 2,
		},
		{
			name:    "trivial true expression",
			raw:     `{"always": "true"}`,
			wantLen: 1,
		},

		{
			name:    "malformed JSON",
			raw:     `{ipsec-mtu-fix: bad}`,
			wantErr: "decode predicates flag",
		},
		{
			name:    "wrong JSON shape (top-level array)",
			raw:     `["ipsec-mtu-fix"]`,
			wantErr: "decode predicates flag",
		},
		{
			name:    "invalid workaround name",
			raw:     `{"Bad_Name": "true"}`,
			wantErr: "not a valid workaround name",
		},
		{
			name:    "empty expression",
			raw:     `{"a": ""}`,
			wantErr: "empty expression",
		},
		{
			name:    "CEL syntax error",
			raw:     `{"a": "region ==="}`,
			wantErr: "compile CEL",
		},
		{
			name:    "CEL unknown variable",
			raw:     `{"a": "nonsense == \"x\""}`,
			wantErr: "compile CEL",
		},
		{
			name:    "CEL non-bool return type",
			raw:     `{"a": "region"}`,
			wantErr: "must return bool",
		},
		{
			name:    "expression too long",
			raw:     `{"a": "` + strings.Repeat("x == \\\"y\\\" || ", 1000) + `true"}`,
			wantErr: "exceeds",
		},
		{
			name:    "flag value too large",
			raw:     "{" + strings.Repeat(" ", MaxPredicatesBytes) + "}",
			wantErr: "cap is",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePredicates(tt.raw)
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
			if len(got) != tt.wantLen {
				t.Fatalf("got %d predicates, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestPredicatesEval(t *testing.T) {
	type want struct {
		matched      bool
		hasPredicate bool
		errSubstring string // empty means no error
	}

	tests := []struct {
		name   string
		raw    string
		lookup string
		facts  ClusterFacts
		want   want
	}{
		{
			// The core "this cluster is not opted in" path: a workaround
			// without an entry in the predicates flag must return
			// hasPredicate=false so the controller treats it as disabled.
			name:   "missing entry: hasPredicate false",
			raw:    `{}`,
			lookup: "not-configured",
			facts:  ClusterFacts{},
			want:   want{matched: false, hasPredicate: false},
		},
		{
			name:   "configured entry: matches",
			raw:    `{"ipsec-mtu-fix": "ipsecMode == \"Full\""}`,
			lookup: "ipsec-mtu-fix",
			facts:  ClusterFacts{IPSecMode: "Full"},
			want:   want{matched: true, hasPredicate: true},
		},
		{
			name:   "configured entry: does not match",
			raw:    `{"ipsec-mtu-fix": "ipsecMode == \"Full\""}`,
			lookup: "ipsec-mtu-fix",
			facts:  ClusterFacts{IPSecMode: "External"},
			want:   want{matched: false, hasPredicate: true},
		},
		{
			// `ipsec.mode != "Disabled"` style negated-equality (from the
			// user's sketched design) lights up for every non-Disabled mode.
			name:   "negated equality matches non-Disabled",
			raw:    `{"ipsec-fix": "ipsecMode != \"Disabled\""}`,
			lookup: "ipsec-fix",
			facts:  ClusterFacts{IPSecMode: "Full"},
			want:   want{matched: true, hasPredicate: true},
		},
		{
			name:   "negated equality rejects Disabled",
			raw:    `{"ipsec-fix": "ipsecMode != \"Disabled\""}`,
			lookup: "ipsec-fix",
			facts:  ClusterFacts{IPSecMode: "Disabled"},
			want:   want{matched: false, hasPredicate: true},
		},
		{
			name:   "version range matches",
			raw:    `{"kernel-4-16": "versionAtLeast(clusterVersion, \"4.16.0\") && versionLessThan(clusterVersion, \"4.17.0\")"}`,
			lookup: "kernel-4-16",
			facts:  ClusterFacts{ClusterVersion: mustVersion(t, "4.16.5")},
			want:   want{matched: true, hasPredicate: true},
		},
		{
			name:   "version range outside",
			raw:    `{"kernel-4-16": "versionAtLeast(clusterVersion, \"4.16.0\") && versionLessThan(clusterVersion, \"4.17.0\")"}`,
			lookup: "kernel-4-16",
			facts:  ClusterFacts{ClusterVersion: mustVersion(t, "4.17.3")},
			want:   want{matched: false, hasPredicate: true},
		},
		{
			// Combined: region + version. This is the most common shape.
			name:   "combined region and version",
			raw:    `{"x": "region == \"eastus\" && versionAtLeast(clusterVersion, \"4.16.0\")"}`,
			lookup: "x",
			facts: ClusterFacts{
				ClusterVersion: mustVersion(t, "4.17.0"),
				Location:       "eastus",
			},
			want: want{matched: true, hasPredicate: true},
		},
		{
			// Runtime error: bad version arg propagates through Eval.
			name:   "runtime error propagates",
			raw:    `{"bad": "versionAtLeast(clusterVersion, \"not-a-version\")"}`,
			lookup: "bad",
			facts:  ClusterFacts{ClusterVersion: mustVersion(t, "4.17.0")},
			want:   want{matched: false, hasPredicate: true, errSubstring: "versionAtLeast"},
		},
		{
			// Unknown cluster version + version helper: must NOT silently
			// match. This is the same fail-closed property the unit-level
			// CEL tests cover, exercised through the public surface.
			name:   "unknown cluster version fails closed",
			raw:    `{"x": "versionAtLeast(clusterVersion, \"4.16.0\")"}`,
			lookup: "x",
			facts:  ClusterFacts{ClusterVersion: nil},
			want:   want{matched: false, hasPredicate: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preds, err := parsePredicates(tt.raw)
			if err != nil {
				t.Fatalf("parsePredicates(%q): %v", tt.raw, err)
			}
			matched, hasPredicate, err := preds.Eval(context.Background(), tt.lookup, tt.facts)
			switch {
			case tt.want.errSubstring == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.want.errSubstring != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.want.errSubstring)
			case tt.want.errSubstring != "" && !strings.Contains(err.Error(), tt.want.errSubstring):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.want.errSubstring)
			}
			if matched != tt.want.matched {
				t.Errorf("matched = %v, want %v", matched, tt.want.matched)
			}
			if hasPredicate != tt.want.hasPredicate {
				t.Errorf("hasPredicate = %v, want %v", hasPredicate, tt.want.hasPredicate)
			}
		})
	}
}

// TestPredicatesEvalNilReceiver guards a future refactor: a nil Predicates
// map must behave the same as an empty one (hasPredicate=false for every
// name) so callers don't have to special-case the disabled cluster path.
func TestPredicatesEvalNilReceiver(t *testing.T) {
	var p Predicates
	matched, hasPredicate, err := p.Eval(context.Background(), "anything", ClusterFacts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched || hasPredicate {
		t.Fatalf("nil Predicates: matched=%v hasPredicate=%v; both want false", matched, hasPredicate)
	}
}
