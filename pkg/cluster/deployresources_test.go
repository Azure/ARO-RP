package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/types"
	aztypes "github.com/openshift/installer/pkg/types/azure"
)

func TestZones(t *testing.T) {
	for _, tt := range []struct {
		name    string
		zones   []string
		want    *[]string
		wantErr string
	}{
		{
			name:  "no zones, 3 replicas",
			zones: []string{""},
		},
		{
			name:    "1 zone, 3 replicas",
			zones:   []string{"1"},
			wantErr: "cluster creation with 1 zone(s) and 3 replica(s) is unimplemented",
		},
		{
			name:    "2 zones, 3 replicas",
			zones:   []string{"1", "2"},
			wantErr: "cluster creation with 2 zone(s) and 3 replica(s) is unimplemented",
		},
		{
			name:  "3 zones, 3 replicas",
			zones: []string{"1", "2", "3"},
			want: &[]string{
				"[copyIndex(1)]",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zones, err := zones(&installconfig.InstallConfig{
				Config: &types.InstallConfig{
					ControlPlane: &types.MachinePool{
						Platform: types.MachinePoolPlatform{
							Azure: &aztypes.MachinePool{
								Zones: tt.zones,
							},
						},
						Replicas: to.Int64Ptr(3),
					},
				},
			})
			if err != nil && tt.wantErr != err.Error() ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
			if !reflect.DeepEqual(tt.want, zones) {
				t.Error(zones)
			}
		})
	}
}
