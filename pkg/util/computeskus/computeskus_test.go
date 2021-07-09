package computeskus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestZones(t *testing.T) {
	for _, tt := range []struct {
		name      string
		sku       *mgmtcompute.ResourceSku
		wantZones []string
	}{
		{
			name: "sku with location info present",
			sku: &mgmtcompute.ResourceSku{
				LocationInfo: &([]mgmtcompute.ResourceSkuLocationInfo{
					{Zones: &([]string{"1", "2", "3"})},
				}),
			},
			wantZones: []string{"1", "2", "3"},
		},
		{
			name: "sku with location info present, but zones field is nil",
			sku: &mgmtcompute.ResourceSku{
				LocationInfo: &([]mgmtcompute.ResourceSkuLocationInfo{
					{Zones: nil},
				}),
			},
		},
		{
			name: "sku with location info present, but empty",
			sku: &mgmtcompute.ResourceSku{
				LocationInfo: &([]mgmtcompute.ResourceSkuLocationInfo{}),
			},
		},
		{
			name: "sku with location info missing",
			sku:  &mgmtcompute.ResourceSku{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zones := Zones(tt.sku)

			if !reflect.DeepEqual(tt.wantZones, zones) {
				t.Error(cmp.Diff(tt.wantZones, zones))
			}
		})
	}
}

func TestHasCapability(t *testing.T) {
	fakeCapabilityName := "fakeCapability"

	for _, tt := range []struct {
		name       string
		sku        *mgmtcompute.ResourceSku
		wantResult bool
	}{
		{
			name: "sku explicitly supports capability",
			sku: &mgmtcompute.ResourceSku{
				Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
					{Name: &fakeCapabilityName, Value: to.StringPtr("True")},
				}),
			},
			wantResult: true,
		},
		{
			name: "sku explicitly does not support capability",
			sku: &mgmtcompute.ResourceSku{
				Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{
					{Name: &fakeCapabilityName, Value: to.StringPtr("False")},
				}),
			},
		},
		{
			name: "sku implicitly does not support capability because it is missing from the list",
			sku: &mgmtcompute.ResourceSku{
				Capabilities: &([]mgmtcompute.ResourceSkuCapabilities{}),
			},
		},
		{
			name: "sku implicitly does not support capability, because capabilities info missing",
			sku:  &mgmtcompute.ResourceSku{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := HasCapability(tt.sku, fakeCapabilityName)

			if result != tt.wantResult {
				t.Error(result)
			}
		})
	}
}
