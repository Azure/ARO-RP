package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
)

func TestParseOverrideClusterDeploymentZones(t *testing.T) {
	for _, tt := range []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "unset", raw: "", want: nil},
		{name: "whitespace only", raw: "   ", want: nil},
		{name: "nonzonal", raw: "nonzonal", want: []string{}},
		{name: "nonzonal mixed case", raw: "NonZonal", want: []string{}},
		{name: "nonzonal padded", raw: " nonzonal ", want: []string{}},
		{name: "single zone", raw: "1", want: []string{"1"}},
		{name: "three zones", raw: "1,2,3", want: []string{"1", "2", "3"}},
		{name: "trims spaces", raw: " 1, 2 ,3", want: []string{"1", "2", "3"}},
		{name: "two zones unsupported", raw: "1,2", wantErr: true},
		{name: "zone 4 out of range", raw: "4", wantErr: true},
		{name: "duplicate", raw: "1,1,1", wantErr: true},
		{name: "out of range", raw: "5", wantErr: true},
		{name: "zero", raw: "0", wantErr: true},
		{name: "non-numeric", raw: "eastus-az1", wantErr: true},
		{name: "empty entry", raw: "1,,3", wantErr: true},
		{name: "trailing comma", raw: "1,2,3,", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOverrideClusterDeploymentZones(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %v", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseOverrideClusterDeploymentZones(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}
