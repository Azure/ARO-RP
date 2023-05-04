package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateVMSku(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		restrictions          mgmtcompute.ResourceSkuRestrictionsReasonCode
		restrictionLocation   *[]string
		restrictedZones       []string
		workerProfile1Sku     string
		workerProfile2Sku     string
		masterProfileSku      string
		availableSku          string
		availableSku2         string
		restrictedSku         string
		resourceSkusClientErr error
		wantErr               string
	}{
		{
			name:              "worker and master skus are valid",
			workerProfile1Sku: "Standard_D4s_v2",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_D4s_v2",
		},
		{
			name:              "worker and master skus are distinct, both valid",
			workerProfile1Sku: "Standard_E104i_v5",
			workerProfile2Sku: "Standard_E104i_v5",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_E104i_v5",
			availableSku2:     "Standard_D4s_v2",
		},
		{
			name:              "worker and master skus are distinct, one invalid",
			workerProfile1Sku: "Standard_E104i_v5",
			workerProfile2Sku: "Standard_E104i_v5",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_E104i_v5",
			availableSku2:     "Standard_E104i_v5",
			wantErr:           "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_D4s_v2' is unavailable in region 'eastus'",
		},
		{
			name:              "worker and master skus are distinct, both invalid",
			workerProfile1Sku: "Standard_E104i_v5",
			workerProfile2Sku: "Standard_E104i_v5",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_L8s_v2",
			availableSku2:     "Standard_L16s_v2",
			wantErr:           "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_D4s_v2' is unavailable in region 'eastus'",
		},
		{
			name:                  "unable to retrieve skus information",
			workerProfile1Sku:     "Standard_D4s_v2",
			workerProfile2Sku:     "Standard_D4s_v2",
			resourceSkusClientErr: errors.New("unable to retrieve skus information"),
			wantErr:               "unable to retrieve skus information",
		},
		{
			name:              "desired worker sku doesn't exist in the target region",
			workerProfile1Sku: "Standard_L80",
			workerProfile2Sku: "Standard_L80",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_D4s_v2",
			wantErr:           "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
		},
		{
			name:              "desired master sku doesn't exist in the target region",
			workerProfile1Sku: "Standard_D4s_v2",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_L80",
			availableSku:      "Standard_D4s_v2",
			wantErr:           "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
		},
		{
			name:              "one valid workerprofile and one invalid workerprofile",
			workerProfile1Sku: "Standard_L80",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_D4s_v2",
			wantErr:           "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
		},
		{
			name:         "worker sku exists in region but is not available in subscription",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			workerProfile1Sku: "Standard_L80",
			workerProfile2Sku: "Standard_L80",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_D4s_v2",
			restrictedSku:     "Standard_L80",
			wantErr:           "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
		{
			name:         "master sku exists in region but is not available in subscription",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			workerProfile1Sku: "Standard_D4s_v2",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_L80",
			availableSku:      "Standard_D4s_v2",
			restrictedSku:     "Standard_L80",
			wantErr:           "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
		{
			name:         "sku is restricted in a single zone",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			restrictedZones:   []string{"3"},
			workerProfile1Sku: "Standard_D4s_v2",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_L80",
			availableSku:      "Standard_D4s_v2",
			restrictedSku:     "Standard_L80",
			wantErr:           "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.restrictedZones == nil {
				tt.restrictedZones = []string{"1", "2", "3"}
			}

			controller := gomock.NewController(t)
			defer controller.Finish()

			oc := &api.OpenShiftCluster{
				Location: "eastus",
				Properties: api.OpenShiftClusterProperties{
					WorkerProfiles: []api.WorkerProfile{
						{
							VMSize: api.VMSize(tt.workerProfile1Sku),
						},
						{
							VMSize: api.VMSize(tt.workerProfile2Sku),
						},
					},
					MasterProfile: api.MasterProfile{
						VMSize: api.VMSize(tt.masterProfileSku),
					},
				},
			}

			skus := []mgmtcompute.ResourceSku{
				{
					Name:      &tt.availableSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &[]string{"1, 2, 3"}},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{},
					ResourceType: to.StringPtr("virtualMachines"),
				},
				{
					Name:      &tt.availableSku2,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &[]string{"1, 2, 3"}},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{},
					ResourceType: to.StringPtr("virtualMachines"),
				},
				{
					Name:      &tt.restrictedSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &tt.restrictedZones},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{
						{
							ReasonCode: tt.restrictions,
							RestrictionInfo: &mgmtcompute.ResourceSkuRestrictionInfo{
								Locations: tt.restrictionLocation,
							},
						},
					},
					Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{},
					ResourceType: to.StringPtr("virtualMachines"),
				},
			}

			resourceSkusClient := mock_compute.NewMockResourceSkusClient(controller)
			resourceSkusClient.EXPECT().
				List(gomock.Any(), fmt.Sprintf("location eq %v", "eastus")).
				Return(skus, tt.resourceSkusClientErr)

			err := validateVMSku(context.Background(), oc, resourceSkusClient)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
