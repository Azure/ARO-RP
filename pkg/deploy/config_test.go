package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
)

func TestMergeConfig(t *testing.T) {
	for _, tt := range []struct {
		name      string
		primary   Configuration
		secondary Configuration
		want      Configuration
	}{
		{
			name: "noop",
		},
		{
			name: "overrides",
			primary: Configuration{
				DatabaseAccountName:    "primary accountname",
				FPServerCertCommonName: "primary fpcert",
			},
			secondary: Configuration{
				FPServerCertCommonName: "secondary fpcert",
				KeyvaultPrefix:         "secondary kv",
			},
			want: Configuration{
				DatabaseAccountName:    "primary accountname",
				FPServerCertCommonName: "primary fpcert",
				KeyvaultPrefix:         "secondary kv",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeConfig(&tt.primary, &tt.secondary)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(&tt.want, got) {
				t.Fatalf("%#v", got)
			}
		})
	}
}
