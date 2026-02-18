package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/vms"
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
		name          string
		vmSize        vms.VMSize
		isMaster      bool
		desiredResult bool
	}{
		{
			name:          "vmSize is supported for use in ARO as worker node",
			vmSize:        vms.VMSizeStandardF72sV2,
			isMaster:      false,
			desiredResult: true,
		},
		{
			name:   "vmSize is not supported for use in ARO as worker node",
			vmSize: vms.VMSize("Unsupported_Csv_v6"),

			isMaster:      false,
			desiredResult: false,
		},
		{
			name:   "vmSize is supported for use in ARO as master node",
			vmSize: vms.VMSizeStandardF72sV2,

			isMaster:      true,
			desiredResult: true,
		},
		{
			name:   "vmSize is not supported for use in ARO as master node",
			vmSize: vms.VMSizeStandardD2sV3,

			isMaster:      true,
			desiredResult: false,
		},
		{
			name:   "Lsv4 vmSize is supported for use in ARO as worker node",
			vmSize: vms.VMSizeStandardL8sV4,

			isMaster:      false,
			desiredResult: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := VMSizeIsValid(tt.vmSize, tt.isMaster)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}

func TestVMSizeIsValidForVersion(t *testing.T) {
	for _, tt := range []struct {
		name          string
		vmSize        vms.VMSize
		isMaster      bool
		version       string
		desiredResult bool
	}{
		// 4.19+ Master/Control Plane VM sizes - DSv6 series
		{
			name:   "Standard_D8s_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16s_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD16sV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32s_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD32sV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64s_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD64sV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96s_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD96sV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Master/Control Plane VM sizes - DDSv6 series
		{
			name:   "Standard_D8ds_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD8dsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16ds_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD16dsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32ds_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD32dsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64ds_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD64dsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96ds_v6 is valid for 4.19 master",
			vmSize: vms.VMSizeStandardD96dsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DSv6 series
		{
			name:   "Standard_D8s_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16s_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD16sV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32s_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD32sV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64s_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD64sV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96s_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD96sV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DDSv6 series
		{
			name:   "Standard_D8ds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD8dsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16ds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD16dsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32ds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD32dsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64ds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD64dsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96ds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD96dsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DLSv6 series (worker only)
		{
			name:   "Standard_D4ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD4lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D8ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD8lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD16lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD32lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D48ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD48lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD64lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96ls_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD96lsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DLDSv6 series (worker only)
		{
			name:   "Standard_D4lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD4ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D8lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD8ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D16lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD16ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D32lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD32ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D48lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD48ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D64lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD64ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_D96lds_v6 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardD96ldsV6,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - LSv4 series
		{
			name:   "Standard_L8s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL8sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_L16s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL16sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_L32s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL32sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_L48s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL48sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_L64s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL64sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:   "Standard_L80s_v4 is valid for 4.19 worker",
			vmSize: vms.VMSizeStandardL80sV4,

			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// DLSv6 and DLDSv6 are not supported for master/control plane
		{
			name:   "Standard_D4ls_v6 is not valid for 4.19 master",
			vmSize: vms.VMSizeStandardD4lsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: false,
		},
		{
			name:   "Standard_D4lds_v6 is not valid for 4.19 master",
			vmSize: vms.VMSizeStandardD4ldsV6,

			isMaster:      true,
			version:       "4.19.0",
			desiredResult: false,
		},
		// Test older versions (< 4.19) - should not support new v6 instances directly
		// Note: These fall back to VMSizeIsValid() which includes all supported sizes
		{
			name:   "Standard_D8s_v6 falls back to standard validation for 4.18 master",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "4.18.0",
			desiredResult: false,
		},
		{
			name:   "Standard_D8s_v6 falls back to standard validation for 4.18 worker",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
		// Test version edge cases
		{
			name:   "Standard_D8s_v6 is valid for 4.19.1 master",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "4.19.1",
			desiredResult: true,
		},
		{
			name:   "Standard_D8s_v6 is valid for 4.20.0 master",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "4.20.0",
			desiredResult: true,
		},
		// Test invalid version strings
		{
			name:   "Standard_D8s_v6 with invalid version falls back to old validation",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "invalid.version",
			desiredResult: false,
		},
		{
			name:   "Standard_D8s_v6 with empty version falls back to old validation",
			vmSize: vms.VMSizeStandardD8sV6,

			isMaster:      true,
			version:       "",
			desiredResult: false,
		},
		// Test existing VM sizes still work with version validation
		{
			name:   "Standard_D8s_v5 is valid for any version as master",
			vmSize: vms.VMSizeStandardD8sV5,

			isMaster:      true,
			version:       "4.18.0",
			desiredResult: true,
		},
		{
			name:   "Standard_F72s_v2 is valid for any version as worker",
			vmSize: vms.VMSizeStandardF72sV2,

			isMaster:      false,
			version:       "4.18.0",
			desiredResult: true,
		},
		// Test LSv4 instances with older versions (< 4.19) - should not be supported
		{
			name:   "Standard_L8s_v4 falls back to standard validation for 4.18 worker",
			vmSize: vms.VMSizeStandardL8sV4,

			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
		{
			name:   "Standard_L80s_v4 falls back to standard validation for 4.18 worker",
			vmSize: vms.VMSizeStandardL80sV4,

			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := VMSizeIsValidForVersion(tt.vmSize, tt.isMaster, tt.version)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}
