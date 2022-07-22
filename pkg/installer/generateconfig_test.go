package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestSupportedDefaultDisk(t *testing.T) {
	for _, tt := range []struct {
		vmSku               string
		supportsPremiumDisk string
		disk                string
	}{
		{
			"Standard_E64is_v3",
			"True",
			"Premium_LRS",
		},
		{
			"Standard_E64i_v3",
			"False",
			"StandardSSD_LRS",
		},
	} {
		t.Run(tt.vmSku, func(t *testing.T) {
			resourceSku := &mgmtcompute.ResourceSku{
				Name: &tt.vmSku,
				Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{
					{
						Name:  to.StringPtr("PremiumIO"),
						Value: &tt.supportsPremiumDisk,
					},
				},
			}

			result := supportedDefaultDisk(resourceSku)
			if result != tt.disk {
				t.Errorf("got %v but want %v", result, tt.disk)
			}
		})
	}
}
