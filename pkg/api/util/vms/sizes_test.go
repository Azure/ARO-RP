package vms

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestMinMasterVMSizesAreSupported(t *testing.T) {
	for size := range MinMasterVMSizes {
		if _, ok := SupportedMasterVmSizes[size]; !ok {
			t.Errorf("MinMasterVMSizes entry %s is not in SupportedMasterVmSizes", size)
		}
	}
}

func TestMinWorkerVMSizesAreSupported(t *testing.T) {
	for size := range MinWorkerVMSizes {
		if _, ok := SupportedWorkerVmSizesForInternalUser[size]; !ok {
			t.Errorf("MinWorkerVMSizes entry %s is not in SupportedWorkerVmSizesForInternalUser", size)
		}
	}
}

func TestMinVMSizesForRoleSorting(t *testing.T) {
	masterSizes := MinVMSizesForRole(VMRoleMaster)
	if len(masterSizes) == 0 {
		t.Fatal("MinVMSizesForRole(master) returned empty slice")
	}
	if len(masterSizes) != len(MinMasterVMSizes) {
		t.Errorf("MinVMSizesForRole(master) returned %d entries, want %d", len(masterSizes), len(MinMasterVMSizes))
	}

	workerSizes := MinVMSizesForRole(VMRoleWorker)
	if len(workerSizes) == 0 {
		t.Fatal("MinVMSizesForRole(worker) returned empty slice")
	}
	if len(workerSizes) != len(MinWorkerVMSizes) {
		t.Errorf("MinVMSizesForRole(worker) returned %d entries, want %d", len(workerSizes), len(MinWorkerVMSizes))
	}

	// Verify D2 sizes (2 cores) come before D4 sizes (4 cores) in worker list
	lastD2Idx := -1
	firstD4Idx := len(workerSizes)
	for i, sz := range workerSizes {
		info := MinWorkerVMSizes[sz]
		if info.CoreCount == 2 && i > lastD2Idx {
			lastD2Idx = i
		}
		if info.CoreCount == 4 && i < firstD4Idx {
			firstD4Idx = i
		}
	}
	if lastD2Idx >= firstD4Idx {
		t.Errorf("D2 sizes should come before D4 sizes in worker list, but last D2 at index %d >= first D4 at index %d", lastD2Idx, firstD4Idx)
	}

	// Verify unknown role returns nil
	if sizes := MinVMSizesForRole("unknown"); sizes != nil {
		t.Errorf("MinVMSizesForRole(unknown) = %v, want nil", sizes)
	}
}

func TestMinVMSizesForRoleWorkerContainsD2(t *testing.T) {
	workerSizes := MinVMSizesForRole(VMRoleWorker)
	d2Found := false
	for _, sz := range workerSizes {
		if sz == VMSizeStandardD2sV3 || sz == VMSizeStandardD2sV4 || sz == VMSizeStandardD2sV5 {
			d2Found = true
			break
		}
	}
	if !d2Found {
		t.Error("MinVMSizesForRole(worker) should contain at least one D2s size")
	}
}
