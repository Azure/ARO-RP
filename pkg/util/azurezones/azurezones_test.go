package azurezones

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestDetermineZones(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		controlPlaneSkuZones  []string
		workerSkuZones        []string
		wantControlPlaneZones []string
		wantWorkerZones       []string
		wantPIPZones          []string
		allowExpandedAZs      bool

		// wantErr matches the sentinel error wrapped by the returned ZoneError.
		wantErr error
		// wantProfile, wantSKUZones and wantSupportedZones check the facts
		// carried by the ZoneError.
		wantProfile        VMRole
		wantSKUZones       int
		wantSupportedZones int
	}{
		{
			name:                 "control plane in 0 zones, zonal workers",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              ErrMixedZonal,
			wantProfile:          VMRoleMaster,
		},
		{
			name:                  "non-zonal",
			controlPlaneSkuZones:  []string{},
			workerSkuZones:        []string{},
			wantControlPlaneZones: []string{},
			wantWorkerZones:       []string{},
			wantPIPZones:          []string{},
		},
		{
			name:                  "non-zonal but API returns a nil slice",
			controlPlaneSkuZones:  nil,
			workerSkuZones:        nil,
			wantControlPlaneZones: []string{},
			wantWorkerZones:       []string{},
			wantPIPZones:          []string{},
		},
		{
			name:                 "non-zonal control plane, zonal workers",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              ErrMixedZonal,
			wantProfile:          VMRoleMaster,
		},
		{
			name:                 "zonal control plane, non-zonal workers",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{},
			wantErr:              ErrMixedZonal,
			wantProfile:          VMRoleWorker,
		},
		{
			name:                  "zonal control plane, zonal workers",
			controlPlaneSkuZones:  []string{"1", "2", "3"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantControlPlaneZones: []string{"1", "2", "3"},
			wantWorkerZones:       []string{"1", "2", "3"},
			wantPIPZones:          []string{"1", "2", "3"},
		},
		{
			name:                  "region with 4 availability zones, expanded AZs, control plane uses first 3, workers use all",
			allowExpandedAZs:      true,
			controlPlaneSkuZones:  []string{"1", "2", "3", "4"},
			workerSkuZones:        []string{"1", "2", "3", "4"},
			wantControlPlaneZones: []string{"1", "2", "3"},
			wantWorkerZones:       []string{"1", "2", "3", "4"},
			wantPIPZones:          []string{"1", "2", "3", "4"},
		},
		{
			name:                  "region with 4 availability zones, basic AZs only, control plane and workers use 3",
			allowExpandedAZs:      false,
			controlPlaneSkuZones:  []string{"1", "2", "3", "4"},
			workerSkuZones:        []string{"1", "2", "3", "4"},
			wantControlPlaneZones: []string{"1", "2", "3"},
			wantWorkerZones:       []string{"1", "2", "3"},
			wantPIPZones:          []string{"1", "2", "3"},
		},
		{
			name:                 "not enough control plane zones",
			controlPlaneSkuZones: []string{"1", "2"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         2,
			wantSupportedZones:   2,
		},
		{
			name:                 "not enough control plane zones, basic AZs only",
			controlPlaneSkuZones: []string{"1", "2", "4"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         3,
			wantSupportedZones:   2,
		},
		{
			name:                 "not enough worker zones",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleWorker,
			wantSKUZones:         2,
			wantSupportedZones:   2,
		},
		{
			name:                 "not enough worker zones, basic AZs only",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2", "4"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleWorker,
			wantSKUZones:         3,
			wantSupportedZones:   2,
		},
		{
			// A SKU available only in zones ARO does not support is still zonal.
			// It must not be reported as a mix of zonal and non-zonal SKUs.
			name:                 "control plane only in unsupported zones, zonal workers",
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         3,
		},
		{
			name:                 "worker only in unsupported zones, zonal control plane",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"4", "5", "6"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleWorker,
			wantSKUZones:         3,
		},
		{
			// Asymmetric: the control plane is zonal but only in unsupported
			// zones, the worker is genuinely non-zonal. This must report an
			// insufficient-zones error on the control plane, not a mixed error:
			// "select SKUs that are either both zonal or both non-zonal" is not
			// actionable here.
			name:                 "control plane only in unsupported zones, non-zonal workers",
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         3,
		},
		{
			name:                 "worker only in unsupported zones, non-zonal control plane",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"4", "5", "6"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleWorker,
			wantSKUZones:         3,
		},
		{
			// Both sides filtered away: ARO-supported zones are missing on
			// both, but neither is non-zonal, so this is not a mixed error.
			name:                 "both only in unsupported zones",
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{"4", "5", "6"},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         3,
		},
		{
			name:                  "region with 4 availability zones, expanded AZs, control plane only available in non-consecutive 3, workers use all",
			allowExpandedAZs:      true,
			controlPlaneSkuZones:  []string{"1", "2", "4"},
			workerSkuZones:        []string{"1", "2", "3", "4"},
			wantControlPlaneZones: []string{"1", "2", "4"},
			wantWorkerZones:       []string{"1", "2", "3", "4"},
			wantPIPZones:          []string{"1", "2", "3", "4"},
		},
		{
			// The sufficiency checks run before the mixed check, so a control
			// plane short of zones is reported as insufficient even when the
			// worker is non-zonal. This is the only case that exercises the
			// "N of the 3 required availability zones" branch with a non-zero
			// supported count.
			name:                 "not enough control plane zones, non-zonal workers",
			controlPlaneSkuZones: []string{"1", "2"},
			workerSkuZones:       []string{},
			wantErr:              ErrInsufficientZones,
			wantProfile:          VMRoleMaster,
			wantSKUZones:         2,
			wantSupportedZones:   2,
		},
		{
			// With expanded AZs on, zones 4,5,6 are supported, so the control
			// plane is zonal and the worker is not: a genuine mixed error.
			name:                 "expanded AZs on, zonal control plane and non-zonal workers",
			allowExpandedAZs:     true,
			controlPlaneSkuZones: []string{"4", "5", "6"},
			workerSkuZones:       []string{},
			wantErr:              ErrMixedZonal,
			wantProfile:          VMRoleWorker,
		},
		{
			// With expanded AZs on, a SKU in zones 4,5,6 is usable.
			name:                  "unsupported zones are usable when expanded AZs are on",
			allowExpandedAZs:      true,
			controlPlaneSkuZones:  []string{"4", "5", "6"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantControlPlaneZones: []string{"4", "5", "6"},
			wantWorkerZones:       []string{"1", "2", "3"},
			wantPIPZones:          []string{"1", "2", "3", "4", "5", "6"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controlPlaneSku := &armcompute.ResourceSKU{
				Name: new("controlPlaneSKU"),
				LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
					{Zones: pointerutils.ToSlicePtr(tt.controlPlaneSkuZones)},
				}),
			}
			workerSku := &armcompute.ResourceSKU{
				Name: new("workerSKU"),
				LocationInfo: pointerutils.ToSlicePtr([]armcompute.ResourceSKULocationInfo{
					{Zones: pointerutils.ToSlicePtr(tt.workerSkuZones)},
				}),
			}

			m := NewManager(tt.allowExpandedAZs)

			controlPlaneZones, workerZones, PIPZones, err := m.DetermineAvailabilityZones(controlPlaneSku, workerSku)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got error %v, want %v", err, tt.wantErr)
				}
				var zoneErr *ZoneError
				if !errors.As(err, &zoneErr) {
					t.Fatalf("expected *ZoneError, got %T", err)
				}
				if zoneErr.Profile != tt.wantProfile {
					t.Errorf("got profile %q, want %q", zoneErr.Profile, tt.wantProfile)
				}
				if zoneErr.SKUZones != tt.wantSKUZones {
					t.Errorf("got SKUZones %d, want %d", zoneErr.SKUZones, tt.wantSKUZones)
				}
				if zoneErr.SupportedZones != tt.wantSupportedZones {
					t.Errorf("got SupportedZones %d, want %d", zoneErr.SupportedZones, tt.wantSupportedZones)
				}
			}

			if !reflect.DeepEqual(controlPlaneZones, tt.wantControlPlaneZones) {
				t.Error("controlPlaneZones", cmp.Diff(tt.wantControlPlaneZones, controlPlaneZones))
			}

			if !reflect.DeepEqual(workerZones, tt.wantWorkerZones) {
				t.Error("workerZones", cmp.Diff(tt.wantWorkerZones, workerZones))
			}

			if !reflect.DeepEqual(PIPZones, tt.wantPIPZones) {
				t.Error("PIPZones", cmp.Diff(tt.wantPIPZones, PIPZones))
			}
		})
	}
}
