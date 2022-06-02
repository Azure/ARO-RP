package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestDiskSizeIsValid(t *testing.T) {
	for _, tt := range []struct {
		name          string
		diskSize      int
		desiredResult bool
	}{
		{
			name:          "size is valid",
			diskSize:      129,
			desiredResult: true,
		},
		{
			name:          "size is not valid",
			diskSize:      127,
			desiredResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := DiskSizeIsValid(tt.diskSize)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}

func TestVMSizeIsValid(t *testing.T) {
	for _, tt := range []struct {
		name                string
		vmSize              api.VMSize
		requireD2sV3Workers bool
		isMaster            bool
		desiredResult       bool
	}{
		{
			name:                "vmSize is supported for use in ARO as worker node",
			vmSize:              api.VMSizeStandardF72sV2,
			requireD2sV3Workers: false,
			isMaster:            false,
			desiredResult:       true,
		},
		{
			name:                "vmSize is not supported for use in ARO as worker node",
			vmSize:              api.VMSize("Unsupported_Csv_v6"),
			requireD2sV3Workers: false,
			isMaster:            false,
			desiredResult:       false,
		},
		{
			name:                "vmSize is supported for use in ARO as master node",
			vmSize:              api.VMSizeStandardF72sV2,
			requireD2sV3Workers: false,
			isMaster:            true,
			desiredResult:       true,
		},
		{
			name:                "vmSize is not supported for use in ARO as master node",
			vmSize:              api.VMSizeStandardD2sV3,
			requireD2sV3Workers: false,
			isMaster:            true,
			desiredResult:       false,
		},
		{
			name:                "install requires Standard_D2s_v3 workers, worker vmSize is not Standard_D2s_v3",
			vmSize:              api.VMSizeStandardF72sV2,
			requireD2sV3Workers: true,
			isMaster:            false,
			desiredResult:       false,
		},
		{
			name:                "install requires Standard_D2s_v3 workers, worker vmSize is Standard_D2s_v3",
			vmSize:              api.VMSizeStandardD2sV3,
			requireD2sV3Workers: true,
			isMaster:            false,
			desiredResult:       true,
		},
		{
			name:                "install requires Standard_D2s_v3 workers, vmSize is is a master",
			vmSize:              api.VMSizeStandardF72sV2,
			requireD2sV3Workers: true,
			isMaster:            true,
			desiredResult:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := VMSizeIsValid(tt.vmSize, tt.requireD2sV3Workers, tt.isMaster)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}
