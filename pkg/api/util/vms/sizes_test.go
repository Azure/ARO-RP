package vms

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestMinMasterVMSizesAreSupported(t *testing.T) {
	for size := range minMasterVMSizes {
		if _, ok := SupportedMasterVMSizes[size]; !ok {
			t.Errorf("minMasterVMSizes entry %s is not in SupportedMasterVMSizes", size)
		}
	}
}

func TestMinWorkerVMSizesAreSupported(t *testing.T) {
	for size := range minWorkerVMSizes {
		if _, ok := SupportedWorkerVMSizesForTesting[size]; !ok {
			t.Errorf("minWorkerVMSizes entry %s is not in supportedWorkerVMSizesForInternalUser", size)
		}
	}
}

func TestLookupVMSizeFindsKnownAndUnknownSizes(t *testing.T) {
	tests := []struct {
		name             string
		vmSize           VMSize
		wantFound        bool
		wantCoreCount    int
		wantMinVersion19 bool
	}{
		{
			name:          "finds production worker size",
			vmSize:        VMSizeStandardD4sV3,
			wantFound:     true,
			wantCoreCount: 4,
		},
		{
			name:             "finds CI-only worker size with minimum version",
			vmSize:           VMSizeStandardD2sV6,
			wantFound:        true,
			wantCoreCount:    2,
			wantMinVersion19: true,
		},
		{
			name:          "finds CI-only master size",
			vmSize:        VMSizeStandardD4sV4,
			wantFound:     true,
			wantCoreCount: 4,
		},
		{
			name:      "unknown size is not found",
			vmSize:    VMSize("Standard_NotARealSize"),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := LookupVMSize(tt.vmSize)
			if found != tt.wantFound {
				t.Fatalf("LookupVMSize(%q) found=%v, want %v", tt.vmSize, found, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if got.CoreCount != tt.wantCoreCount {
				t.Fatalf("LookupVMSize(%q) coreCount=%d, want %d", tt.vmSize, got.CoreCount, tt.wantCoreCount)
			}
			if tt.wantMinVersion19 && !got.MinimumVersion.Eq(ver419) {
				t.Fatalf("LookupVMSize(%q) minimumVersion=%v, want %v", tt.vmSize, got.MinimumVersion, ver419)
			}
		})
	}
}

func TestGetCICandidateMasterVMSizesMatchMinimumMasterSet(t *testing.T) {
	assertCandidateSetMatchesMinSizes(t, GetCICandidateMasterVMSizes(), minMasterVMSizes)
}

func TestGetCICandidateWorkerVMSizesMatchMinimumWorkerSet(t *testing.T) {
	assertCandidateSetMatchesMinSizes(t, GetCICandidateWorkerVMSizes(), minWorkerVMSizes)
}

func assertCandidateSetMatchesMinSizes(t *testing.T, candidates []VMSize, expected map[VMSize]VMSizeStruct) {
	t.Helper()

	if len(candidates) != len(expected) {
		t.Fatalf("got %d candidates, want %d", len(candidates), len(expected))
	}

	seen := map[VMSize]bool{}
	lastCoreCount := -1

	for _, candidate := range candidates {
		sizeInfo, ok := expected[candidate]
		if !ok {
			t.Fatalf("candidate %q is not in expected minimum size set", candidate)
		}
		if seen[candidate] {
			t.Fatalf("candidate %q appears more than once", candidate)
		}
		seen[candidate] = true

		if sizeInfo.CoreCount < lastCoreCount {
			t.Fatalf("candidate core counts are not non-decreasing: %d came after %d", sizeInfo.CoreCount, lastCoreCount)
		}
		lastCoreCount = sizeInfo.CoreCount
	}

	for candidate := range expected {
		if !seen[candidate] {
			t.Fatalf("expected candidate %q was not returned", candidate)
		}
	}
}
