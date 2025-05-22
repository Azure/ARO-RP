package azurezones

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/go-cmp/cmp"
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
		forceSingleZone       bool
		singleZoneToUse       string
		wantErr               string
	}{
		{
			name:                 "control plane in 0 zones, zonal workers",
			controlPlaneSkuZones: []string{},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              "cluster creation with mix of zonal and non-zonal resources is unsupported (control plane zones: 0, worker zones: 3)",
		},
		{
			name:                  "non-zonal",
			controlPlaneSkuZones:  nil,
			workerSkuZones:        nil,
			wantControlPlaneZones: []string{"", "", ""},
			wantWorkerZones:       []string{""},
			wantPIPZones:          []string{},
		},
		{
			name:                  "force single zone does nothing in non-zonal",
			forceSingleZone:       true,
			singleZoneToUse:       "3",
			controlPlaneSkuZones:  nil,
			workerSkuZones:        nil,
			wantControlPlaneZones: []string{"", "", ""},
			wantWorkerZones:       []string{""},
			wantPIPZones:          []string{},
		},
		{
			name:                 "force single zone, control plane zone not available",
			forceSingleZone:      true,
			singleZoneToUse:      "3",
			controlPlaneSkuZones: []string{"1", "2"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              "control plane SKU 'controlPlaneSKU' is not available in zone '3'",
		},
		{
			name:                 "force single zone, worker zone not available",
			forceSingleZone:      true,
			singleZoneToUse:      "3",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2"},
			wantErr:              "worker SKU 'workerSKU' is not available in zone '3'",
		},
		{
			name:                 "non-zonal control plane, zonal workers",
			controlPlaneSkuZones: nil,
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              "cluster creation with mix of zonal and non-zonal resources is unsupported (control plane zones: 0, worker zones: 3)",
		},
		{
			name:                 "zonal control plane, non-zonal workers",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       nil,
			wantErr:              "cluster creation with mix of zonal and non-zonal resources is unsupported (control plane zones: 3, worker zones: 0)",
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
			name:                  "zonal control plane, zonal workers, forced fixed zone, PIPs are in all",
			forceSingleZone:       true,
			singleZoneToUse:       "3",
			controlPlaneSkuZones:  []string{"1", "2", "3"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantControlPlaneZones: []string{"3", "3", "3"},
			wantWorkerZones:       []string{"3"},
			wantPIPZones:          []string{"1", "2", "3"},
		},
		{
			name:                  "zonal control plane, zonal workers, forced fixed nonzonal",
			forceSingleZone:       true,
			singleZoneToUse:       "",
			controlPlaneSkuZones:  []string{"1", "2", "3"},
			workerSkuZones:        []string{"1", "2", "3"},
			wantControlPlaneZones: []string{"", "", ""},
			wantWorkerZones:       []string{""},
			wantPIPZones:          []string{},
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
			wantErr:              "control plane SKU 'controlPlaneSKU' only available in 2 zones, need 3",
		},
		{
			name:                 "not enough control plane zones, basic AZs only",
			controlPlaneSkuZones: []string{"1", "2", "4"},
			workerSkuZones:       []string{"1", "2", "3"},
			wantErr:              "control plane SKU 'controlPlaneSKU' only available in 2 zones, need 3",
		},
		{
			name:                 "not enough worker zones",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2"},
			wantErr:              "worker SKU 'workerSKU' only available in 2 zones, need 3",
		},
		{
			name:                 "not enough worker zones, basic AZs only",
			controlPlaneSkuZones: []string{"1", "2", "3"},
			workerSkuZones:       []string{"1", "2", "4"},
			wantErr:              "worker SKU 'workerSKU' only available in 2 zones, need 3",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			controlPlaneSku := &mgmtcompute.ResourceSku{
				Name: to.StringPtr("controlPlaneSKU"),
				LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
					{Zones: &tt.controlPlaneSkuZones},
				},
			}
			workerSku := &mgmtcompute.ResourceSku{
				Name: to.StringPtr("workerSKU"),
				LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
					{Zones: &tt.workerSkuZones},
				},
			}

			m := &availabilityZoneManager{
				allowExpandedAvailabilityZones: tt.allowExpandedAZs,
				forceSingleZone:                tt.forceSingleZone,
				singleZoneToUse:                tt.singleZoneToUse,
			}

			controlPlaneZones, workerZones, PIPZones, err := m.DetermineAvailabilityZones(controlPlaneSku, workerSku)
			if err != nil && err.Error() != tt.wantErr {
				t.Error("wantErr", cmp.Diff(tt.wantErr, err))
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
