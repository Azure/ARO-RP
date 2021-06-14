package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"
)

func TestShortCommonName(t *testing.T) {
	tests := []struct {
		name                string
		commonName          string
		wantShortCommonName string
	}{
		{
			name:                "commonName not shortened",
			commonName:          "someShort.common.name.aroapp-dev.something",
			wantShortCommonName: "someShort.common.name.aroapp-dev.something",
		},
		{
			name:                "commonName shortened",
			commonName:          "my-very-long-common-name-needs-shortened.else-things-blow-up.aroapp-dev.something",
			wantShortCommonName: "reserved.aroapp-dev.something",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getShortCommonName(test.commonName)

			if got != test.wantShortCommonName {
				t.Error(fmt.Errorf("got != want: %s != %s", got, test.wantShortCommonName))
			}
		})
	}
}
