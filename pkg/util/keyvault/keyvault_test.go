package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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

func TestCheckOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation *azkeyvault.CertificateOperation
		wantBool  bool
		wantError bool
	}{
		{
			name: "certificate operation is inProgress",
			operation: &azkeyvault.CertificateOperation{
				Status: pointerutils.ToPtr("inProgress"),
			},
			wantBool:  false,
			wantError: false,
		},
		{
			name: "certificate operation is completed",
			operation: &azkeyvault.CertificateOperation{
				Status: pointerutils.ToPtr("completed"),
			},
			wantBool:  true,
			wantError: false,
		},
		{
			name: "certificate operation is failed",
			operation: &azkeyvault.CertificateOperation{
				Status:        pointerutils.ToPtr("failed"),
				StatusDetails: pointerutils.ToPtr("some error"),
				Error: &azkeyvault.Error{
					Code:    pointerutils.ToPtr("some code"),
					Message: pointerutils.ToPtr("some message"),
				},
			},
			wantBool:  false,
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBool, err := checkOperation(test.operation)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.wantBool, gotBool)
		})
	}
}
