package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestRxOpenShiftVersion(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{
			value: "4.3.0",
			want:  true,
		},
		{
			value: "4.3.1",
			want:  true,
		},
		{
			value: "4.3.999",
			want:  true,
		},
		{
			value: "4.3.1000",
		},
		{
			value: "4.3.01",
		},
	} {
		t.Run(tt.value, func(t *testing.T) {
			got := RxOpenShiftVersion.MatchString(tt.value)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}
