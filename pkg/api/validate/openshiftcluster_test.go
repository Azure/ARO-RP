package validate

import (
	"testing"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestOpenShiftClusterName(t *testing.T) {
	clusterName19 := "19characters-aaaaaa"
	clusterName30 := "thisis30characterslong-aaaaaa"

	for _, tt := range []struct {
		name          string
		clusterName   string
		location      string
		desiredResult bool
	}{
		{
			name:          "valid - zoned region > maxLength",
			clusterName:   clusterName30,
			location:      "eastus",
			desiredResult: true,
		},
		{
			name:          "valid - zoned region <= maxLength",
			clusterName:   clusterName19,
			location:      "eastus",
			desiredResult: true,
		},
		{
			name:          "valid - non-zoned region <= maxLength",
			clusterName:   clusterName19,
			location:      "australiasoutheast",
			desiredResult: true,
		},
		{
			name:        "invalid - non-zoned region > maxLength",
			clusterName: clusterName30,
			location:    "australiasoutheast",
		},
		{
			name:        "invalid - non-zoned region > maxLength",
			clusterName: clusterName30,
			location:    "WESTCENTRALUS",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			isValid := OpenShiftClusterNameLength(tt.clusterName, tt.location)
			if isValid != tt.desiredResult {
				t.Errorf("Got %v, wanted %v, for cluster name '%s' in region '%s'", isValid, tt.desiredResult, tt.clusterName, tt.location)
			}
		})
	}
}
