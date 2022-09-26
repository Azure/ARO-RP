package ocm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestValidateOCMFromSystemData(t *testing.T) {
	for _, tt := range []struct {
		name                string
		systemDataHeaderStr string
		validClientIDs      []string
		wantValid           bool
	}{
		{
			name:                "valid system data with matching ID",
			systemDataHeaderStr: `{"lastModifiedBy":"abc-123","lastModifiedByType":"Application"}`,
			validClientIDs:      []string{"abc-123"},
			wantValid:           true,
		},
		{
			name:                "missing lastModifiedByType, invalid",
			systemDataHeaderStr: `{"lastModifiedBy":"abc-123"}`,
			validClientIDs:      []string{},
			wantValid:           false,
		},
		{
			name:                "clientID not in list, invalid",
			systemDataHeaderStr: `{"lastModifiedBy":"abc","lastModifiedByType":"Application"}}`,
			validClientIDs:      []string{"abc-123"},
			wantValid:           false,
		},
		{
			name:                "bad data as header, invalid",
			systemDataHeaderStr: `;;'d;2;l'`,
			validClientIDs:      []string{},
			wantValid:           false,
		},
	} {
		valid := ValidateOCMFromSystemData(tt.systemDataHeaderStr, tt.validClientIDs)

		if tt.wantValid != valid {
			t.Errorf("Expected system data should be %v but got %v", tt.wantValid, valid)
		}
	}
}
