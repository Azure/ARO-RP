package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionZoneMapping(t *testing.T) {
	testCases := []struct {
		desc           string
		location       string
		inputZones     []string
		wantZones      []string
		wantError      error
		mappingProfile *AvailabilityZonesProfile
	}{
		{
			desc:       "happy case",
			location:   "eastus2",
			inputZones: []string{"1", "2", "3"},
			wantZones:  []string{"2", "3", "1"},
			mappingProfile: &AvailabilityZonesProfile{
				Location: "eastus2",
				ZoneMappings: []AvailabilityZoneMapping{
					{
						PhysicalZone: "1",
						LogicalZone:  "2",
					},
					{
						PhysicalZone: "2",
						LogicalZone:  "3",
					},
					{
						PhysicalZone: "3",
						LogicalZone:  "1",
					},
				},
			},
		},
		{
			desc:       "happy case, non-zonal",
			location:   "eastus2",
			inputZones: []string{},
			wantZones:  []string{},
			mappingProfile: &AvailabilityZonesProfile{
				Location: "eastus2",
				ZoneMappings: []AvailabilityZoneMapping{
					{
						PhysicalZone: "1",
						LogicalZone:  "2",
					},
					{
						PhysicalZone: "2",
						LogicalZone:  "3",
					},
					{
						PhysicalZone: "3",
						LogicalZone:  "1",
					},
				},
			},
		},
		{
			desc:           "no matching location",
			location:       "eastus2",
			inputZones:     []string{"1", "2", "3"},
			mappingProfile: nil,
			wantError:      errors.New("unable to map physical zones, subscription has no zone mapping"),
		},
		{
			desc:       "missing zone",
			location:   "eastus2",
			inputZones: []string{"1", "2", "3", "4"},
			mappingProfile: &AvailabilityZonesProfile{
				Location: "eastus2",
				ZoneMappings: []AvailabilityZoneMapping{
					{
						PhysicalZone: "1",
						LogicalZone:  "2",
					},
					{
						PhysicalZone: "2",
						LogicalZone:  "3",
					},
					{
						PhysicalZone: "3",
						LogicalZone:  "1",
					},
				},
			},
			wantError: errors.New("unable to map physical zone '4' to logical zone"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			r := require.New(t)
			sub := &SubscriptionProperties{
				AvailabilityZones: tt.mappingProfile,
			}

			gotZones, err := sub.MapPhysicalZonesToLogicalZonesForSubscription(tt.location, tt.inputZones)

			if tt.wantError != nil {
				r.ErrorContains(err, tt.wantError.Error())
			} else {
				r.Equal(tt.wantZones, gotZones)
			}
		})
	}
}
