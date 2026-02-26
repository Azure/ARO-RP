package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api/util/vms"
)

func TestDiskSizeIsValid(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := DiskSizeIsValid(tt.diskSize)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}

func TestVMSizeIsValid(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name          string
		vmSize        vms.VMSize
		isMaster      bool
		isCI          bool
		desiredResult bool
	}{
		// Production mode (isCI=false) — standard sizes
		{
			name:          "production: supported worker size",
			vmSize:        vms.VMSizeStandardF72sV2,
			isMaster:      false,
			desiredResult: true,
		},
		{
			name:          "production: unsupported worker size",
			vmSize:        vms.VMSize("Unsupported_Csv_v6"),
			isMaster:      false,
			desiredResult: false,
		},
		{
			name:          "production: supported master size (D8+)",
			vmSize:        vms.VMSizeStandardF72sV2,
			isMaster:      true,
			desiredResult: true,
		},
		{
			name:          "production: D2s rejected as master",
			vmSize:        vms.VMSizeStandardD2sV3,
			isMaster:      true,
			desiredResult: false,
		},
		{
			name:          "production: Lsv4 supported as worker",
			vmSize:        vms.VMSizeStandardL8sV4,
			isMaster:      false,
			desiredResult: true,
		},
		// Production mode — CI-only sizes must be REJECTED
		{
			name:          "production: D4s_v3 rejected as master (too small)",
			vmSize:        vms.VMSizeStandardD4sV3,
			isMaster:      true,
			desiredResult: false,
		},
		{
			name:          "production: D2s_v3 rejected as worker (too small)",
			vmSize:        vms.VMSizeStandardD2sV3,
			isMaster:      false,
			desiredResult: false,
		},
		{
			name:          "production: D2s_v5 rejected as worker (too small)",
			vmSize:        vms.VMSizeStandardD2sV5,
			isMaster:      false,
			desiredResult: false,
		},
		// CI mode (isCI=true) — CI-only sizes must be ACCEPTED
		{
			name:          "CI: D4s_v3 accepted as master",
			vmSize:        vms.VMSizeStandardD4sV3,
			isMaster:      true,
			isCI:          true,
			desiredResult: true,
		},
		{
			name:          "CI: D4s_v5 accepted as master",
			vmSize:        vms.VMSizeStandardD4sV5,
			isMaster:      true,
			isCI:          true,
			desiredResult: true,
		},
		{
			name:          "CI: D2s_v3 accepted as worker",
			vmSize:        vms.VMSizeStandardD2sV3,
			isMaster:      false,
			isCI:          true,
			desiredResult: true,
		},
		{
			name:          "CI: D2s_v5 accepted as worker",
			vmSize:        vms.VMSizeStandardD2sV5,
			isMaster:      false,
			isCI:          true,
			desiredResult: true,
		},
		// CI mode — production sizes still accepted
		{
			name:          "CI: D8s_v3 still valid as master",
			vmSize:        vms.VMSizeStandardD8sV3,
			isMaster:      true,
			isCI:          true,
			desiredResult: true,
		},
		{
			name:          "CI: F72s_v2 still valid as worker",
			vmSize:        vms.VMSizeStandardF72sV2,
			isMaster:      false,
			isCI:          true,
			desiredResult: true,
		},
		// CI mode — unsupported sizes still rejected
		{
			name:          "CI: unsupported size rejected as worker",
			vmSize:        vms.VMSize("Unsupported_Csv_v6"),
			isMaster:      false,
			isCI:          true,
			desiredResult: false,
		},
		{
			name:          "CI: D2s_v3 still rejected as master (too small even for CI)",
			vmSize:        vms.VMSizeStandardD2sV3,
			isMaster:      true,
			isCI:          true,
			desiredResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := VMSizeIsValid(tt.vmSize, tt.isMaster, tt.isCI)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}

func TestVMSizeIsValidForVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name          string
		vmSize        vms.VMSize
		isMaster      bool
		version       string
		isCI          bool
		desiredResult bool
	}{
		// 4.19+ Master/Control Plane VM sizes - DSv6 series
		{
			name:          "Standard_D8s_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16s_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD16sV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32s_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD32sV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64s_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD64sV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96s_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD96sV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Master/Control Plane VM sizes - DDSv6 series
		{
			name:          "Standard_D8ds_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD8dsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16ds_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD16dsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32ds_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD32dsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64ds_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD64dsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96ds_v6 is valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD96dsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DSv6 series
		{
			name:          "Standard_D8s_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16s_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD16sV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32s_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD32sV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64s_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD64sV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96s_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD96sV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DDSv6 series
		{
			name:          "Standard_D8ds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD8dsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16ds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD16dsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32ds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD32dsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64ds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD64dsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96ds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD96dsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DLSv6 series (worker only)
		{
			name:          "Standard_D4ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD4lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D8ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD8lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD16lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD32lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D48ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD48lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD64lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96ls_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD96lsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - DLDSv6 series (worker only)
		{
			name:          "Standard_D4lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD4ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D8lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD8ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D16lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD16ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D32lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD32ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D48lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD48ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D64lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD64ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_D96lds_v6 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardD96ldsV6,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// 4.19+ Worker VM sizes - LSv4 series
		{
			name:          "Standard_L8s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL8sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_L16s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL16sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_L32s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL32sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_L48s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL48sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_L64s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL64sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		{
			name:          "Standard_L80s_v4 is valid for 4.19 worker",
			vmSize:        vms.VMSizeStandardL80sV4,
			isMaster:      false,
			version:       "4.19.0",
			desiredResult: true,
		},
		// DLSv6 and DLDSv6 are not supported for master/control plane
		{
			name:          "Standard_D4ls_v6 is not valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD4lsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: false,
		},
		{
			name:          "Standard_D4lds_v6 is not valid for 4.19 master",
			vmSize:        vms.VMSizeStandardD4ldsV6,
			isMaster:      true,
			version:       "4.19.0",
			desiredResult: false,
		},
		// Older versions (< 4.19) reject version-gated sizes
		{
			name:          "Standard_D8s_v6 rejected for 4.18 master",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "4.18.0",
			desiredResult: false,
		},
		{
			name:          "Standard_D8s_v6 rejected for 4.18 worker",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
		// Version edge cases
		{
			name:          "Standard_D8s_v6 is valid for 4.19.1 master",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "4.19.1",
			desiredResult: true,
		},
		{
			name:          "Standard_D8s_v6 is valid for 4.20.0 master",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "4.20.0",
			desiredResult: true,
		},
		// Invalid version strings
		{
			name:          "Standard_D8s_v6 with invalid version string",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "invalid.version",
			desiredResult: false,
		},
		{
			name:          "Standard_D8s_v6 with empty version string",
			vmSize:        vms.VMSizeStandardD8sV6,
			isMaster:      true,
			version:       "",
			desiredResult: false,
		},
		// Existing sizes work with any version
		{
			name:          "Standard_D8s_v5 valid for any version as master",
			vmSize:        vms.VMSizeStandardD8sV5,
			isMaster:      true,
			version:       "4.18.0",
			desiredResult: true,
		},
		{
			name:          "Standard_F72s_v2 valid for any version as worker",
			vmSize:        vms.VMSizeStandardF72sV2,
			isMaster:      false,
			version:       "4.18.0",
			desiredResult: true,
		},
		// LSv4 with older versions
		{
			name:          "Standard_L8s_v4 rejected for 4.18 worker",
			vmSize:        vms.VMSizeStandardL8sV4,
			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
		{
			name:          "Standard_L80s_v4 rejected for 4.18 worker",
			vmSize:        vms.VMSizeStandardL80sV4,
			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
		// CI mode: D4s accepted as master with version check
		{
			name:          "CI: D4s_v3 accepted as master for 4.18",
			vmSize:        vms.VMSizeStandardD4sV3,
			isMaster:      true,
			version:       "4.18.0",
			isCI:          true,
			desiredResult: true,
		},
		{
			name:          "CI: D4s_v5 accepted as master for 4.19",
			vmSize:        vms.VMSizeStandardD4sV5,
			isMaster:      true,
			version:       "4.19.0",
			isCI:          true,
			desiredResult: true,
		},
		// CI mode: D2s accepted as worker with version check
		{
			name:          "CI: D2s_v5 accepted as worker for 4.18",
			vmSize:        vms.VMSizeStandardD2sV5,
			isMaster:      false,
			version:       "4.18.0",
			isCI:          true,
			desiredResult: true,
		},
		// Production mode: D4s/D2s rejected even with valid version
		{
			name:          "production: D4s_v3 rejected as master for 4.18",
			vmSize:        vms.VMSizeStandardD4sV3,
			isMaster:      true,
			version:       "4.18.0",
			desiredResult: false,
		},
		{
			name:          "production: D2s_v5 rejected as worker for 4.18",
			vmSize:        vms.VMSizeStandardD2sV5,
			isMaster:      false,
			version:       "4.18.0",
			desiredResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := VMSizeIsValidForVersion(tt.vmSize, tt.isMaster, tt.version, tt.isCI)

			if result != tt.desiredResult {
				t.Errorf("Want %v, got %v", tt.desiredResult, result)
			}
		})
	}
}
