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

func TestRepoFromReference(t *testing.T) {
	tests := []struct {
		name      string
		reference string
		expected  string
	}{
		{
			name:      "digest reference",
			reference: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:abc123",
			expected:  "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
		},
		{
			name:      "tag reference",
			reference: "quay.io/openshift-release-dev/ocp-release:4.21.7-x86_64",
			expected:  "quay.io/openshift-release-dev/ocp-release",
		},
		{
			name:      "tag reference with port",
			reference: "registry.example.com:5000/image:latest",
			expected:  "registry.example.com:5000/image",
		},
		{
			name:      "no tag or digest",
			reference: "quay.io/openshift-release-dev/ocp-release",
			expected:  "quay.io/openshift-release-dev/ocp-release",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := repoFromReference(test.reference)
			if got != test.expected {
				t.Errorf("got != want: %s != %s", got, test.expected)
			}
		})
	}
}

func TestSigReference(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		digest   string
		expected string
	}{
		{
			name:     "standard sha256 digest",
			repo:     "arosvc.azurecr.io/openshift-release-dev/ocp-release",
			digest:   "sha256:90792dfb2c5ebb89007c02486efca50556a23650be0915c7546fc507ab05e0df",
			expected: "arosvc.azurecr.io/openshift-release-dev/ocp-release:sha256-90792dfb2c5ebb89007c02486efca50556a23650be0915c7546fc507ab05e0df.sig",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sigReference(test.repo, test.digest)
			if got != test.expected {
				t.Errorf("got != want: %s != %s", got, test.expected)
			}
		})
	}
}

func TestDigestFromReference(t *testing.T) {
	tests := []struct {
		name      string
		reference string
		expected  string
		wantErr   bool
	}{
		{
			name:      "digest reference",
			reference: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:abc123def456",
			expected:  "sha256:abc123def456",
		},
		{
			name:      "tag reference returns error",
			reference: "quay.io/openshift-release-dev/ocp-release:4.21.7-x86_64",
			wantErr:   true,
		},
		{
			name:      "no tag or digest returns error",
			reference: "quay.io/openshift-release-dev/ocp-release",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := digestFromReference(test.reference)
			if test.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Errorf("got != want: %s != %s", got, test.expected)
			}
		})
	}
}
