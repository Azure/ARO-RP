package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestManagedDomain(t *testing.T) {
	p := &prod{domain: "eastus.aroapp.io"}

	for _, tt := range []struct {
		domain  string
		want    string
		wantErr string
	}{
		{
			domain: "eastus.aroapp.io",
		},
		{
			domain: "aroapp.io",
		},
		{
			domain: "redhat.com",
		},
		{
			domain: "foo.eastus.aroapp.io.redhat.com",
		},
		{
			domain: "foo.eastus.aroapp.io",
			want:   "foo.eastus.aroapp.io",
		},
		{
			domain: "bar",
			want:   "bar.eastus.aroapp.io",
		},
		{
			domain:  "",
			wantErr: `invalid domain ""`,
		},
		{
			domain:  ".foo",
			wantErr: `invalid domain ".foo"`,
		},
		{
			domain:  "foo.",
			wantErr: `invalid domain "foo."`,
		},
	} {
		t.Run(tt.domain, func(t *testing.T) {
			got, err := p.ManagedDomain(tt.domain)
			if got != tt.want {
				t.Error(got)
			}
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
