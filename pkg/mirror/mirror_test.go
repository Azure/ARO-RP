package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"
)

func TestDestLastIndex(t *testing.T) {
	tests := []struct {
		name                string
		repo                string
		reference           string
		expectedDestination string
	}{
		{
			name:                "DestLastIndex removes path",
			repo:                "destrepo.io",
			reference:           "azurecr.io/some/path/to/image:tag",
			expectedDestination: "destrepo.io/image:tag",
		},
		{
			name:                "DestLastIndex replaces dest acr",
			repo:                "destrepo.io",
			reference:           "azurecr.io/image:tag",
			expectedDestination: "destrepo.io/image:tag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DestLastIndex(test.repo, test.reference)

			if got != test.expectedDestination {
				t.Error(fmt.Errorf("got != want: %s != %s", got, test.expectedDestination))
			}
		})
	}
}

func TestDest(t *testing.T) {
	tests := []struct {
		name                string
		repo                string
		reference           string
		expectedDestination string
	}{
		{
			name:                "Dest Keeps Path",
			repo:                "destrepo.io",
			reference:           "azurecr.io/some/path/to/image:tag",
			expectedDestination: "destrepo.io/some/path/to/image:tag",
		},
		{
			name:                "Dest replaces dest acr",
			repo:                "destrepo.io",
			reference:           "azurecr.io/image:tag",
			expectedDestination: "destrepo.io/image:tag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Dest(test.repo, test.reference)

			if got != test.expectedDestination {
				t.Error(fmt.Errorf("got != want: %s != %s", got, test.expectedDestination))
			}
		})
	}
}
