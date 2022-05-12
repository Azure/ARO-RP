package routefix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestIsRequired(t *testing.T) {
	for _, tt := range []struct {
		name           string
		clusterVersion string
		expectedResult bool
	}{
		{
			name:           "4.4 - Required",
			clusterVersion: "4.4.52",
			expectedResult: true,
		},
		{
			name:           "4.5 - Required",
			clusterVersion: "4.5.10",
			expectedResult: true,
		},
		{
			name:           "4.6 - Required",
			clusterVersion: "4.6.36",
			expectedResult: true,
		},
		{
			name:           "4.6 - Not required",
			clusterVersion: "4.6.37",
			expectedResult: false,
		},
		{
			name:           "4.7 - Required",
			clusterVersion: "4.7.10",
			expectedResult: true,
		},
		{
			name:           "4.7 - Not required",
			clusterVersion: "4.7.18",
			expectedResult: false,
		},
		{
			name:           "4.8 - Not required",
			clusterVersion: "4.8.10",
			expectedResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clusterversion, err := version.ParseVersion(tt.clusterVersion)
			if err != nil {
				t.Errorf("error = %v", err)
				return
			}

			r := Reconciler{
				verFixed46: verFixed46,
				verFixed47: verFixed47,
			}
			if tt.expectedResult != r.isRequired(clusterversion) {
				t.Errorf("Expected %v, but got %v", tt.expectedResult, r.isRequired(clusterversion))
			}
		})
	}
}
