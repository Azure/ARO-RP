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
