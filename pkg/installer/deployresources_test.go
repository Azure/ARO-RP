package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
)

func TestZones(t *testing.T) {
	for _, tt := range []struct {
		name       string
		zones      []string
		region     string
		replicas   int64
		wantMaster *[]string
		wantWorker *[]string
		wantErr    string
	}{
		{
			name:       "no zones, 3 replicas",
			zones:      []string{""},
			wantMaster: nil,
		},
		{
			name:  "1 zone, 3 replicas",
			zones: []string{"1"},
			wantMaster: &[]string{
				"1",
			},
			wantWorker: &[]string{"1"},
		},
		{
			name:  "2 zones, 3 replicas",
			zones: []string{"1", "2"},
			wantMaster: &[]string{
				"1",
				"2",
			},
			wantWorker: &[]string{
				"1",
				"2",
			},
		},
		{
			name:       "centraluseuap",
			zones:      []string{"1", "2"},
			region:     "centraluseuap",
			wantMaster: &[]string{"2"},
			wantWorker: &[]string{"2"},
		},
		{
			name:       "3 zones, 3 replicas",
			zones:      []string{"1", "2", "3"},
			wantMaster: &[]string{"[copyIndex(1)]"},
			wantWorker: &[]string{"[copyIndex(1)]"},
		},
		{
			name:    "4 zones, 3 replicas",
			zones:   []string{"1", "2", "3", "4"},
			wantErr: "cluster creation with 4 zone(s) and 3 replica(s) is unsupported",
		},
		{
			name:     "4 zones, 4 replicas",
			zones:    []string{"1", "2", "3", "4"},
			replicas: 4,
			wantErr:  "cluster creation with 4 zone(s) and 4 replica(s) is unsupported",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.replicas != 4 {
				tt.replicas = 3
			}

			zones, err := zones(&installconfig.InstallConfig{
				Config: &types.InstallConfig{
					ControlPlane: &types.MachinePool{
						Platform: types.MachinePoolPlatform{
							Azure: &azuretypes.MachinePool{
								Zones: tt.zones,
							},
						},
						Replicas: to.Int64Ptr(tt.replicas),
					},
					Platform: types.Platform{
						Azure: &azuretypes.Platform{
							Region: tt.region,
							DefaultMachinePlatform: &azuretypes.MachinePool{
								Zones: tt.zones,
							},
						},
					},
				},
			})
			if err != nil && tt.wantErr != err.Error() ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
			if !reflect.DeepEqual(tt.wantMaster, zones) {
				t.Errorf("Expected master %v, got master %v", tt.wantMaster, zones)
			}
			if !reflect.DeepEqual(tt.wantWorker, zones) {
				t.Errorf("Expected worker %v, got worker %v", tt.wantWorker, zones)
			}
		})
	}
}
