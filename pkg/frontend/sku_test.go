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
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidateVMSku(t *testing.T) {
	for _, tt := range []struct {
		name                       string
		restrictions               mgmtcompute.ResourceSkuRestrictionsReasonCode
		restrictionLocation        *[]string
		restrictedZones            []string
		workerProfile1Sku          string
		workerProfile2Sku          string
		masterProfileSku           string
		availableSku               string
		availableSkuHasEncryption  bool
		availableSku2              string
		availableSku2HasEncryption bool
		restrictedSku              string
		masterEncryptionAtHost     api.EncryptionAtHost
		workerEncryptionAtHost     api.EncryptionAtHost
		resourceSkusClientErr      error
		wpStatus                   bool
		wantErr                    string
	}{
		{
			name:                   "worker and master skus are valid",
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_D4s_v2",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "worker profile is enriched and skus are valid",
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_D4s_v2",
			wpStatus:               true,
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "worker and master skus are distinct, both valid",
			workerProfile1Sku:      "Standard_E104i_v5",
			workerProfile2Sku:      "Standard_E104i_v5",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_E104i_v5",
			availableSku2:          "Standard_D4s_v2",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "worker and master skus are distinct, one invalid",
			workerProfile1Sku:      "Standard_E104i_v5",
			workerProfile2Sku:      "Standard_E104i_v5",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_E104i_v5",
			availableSku2:          "Standard_E104i_v5",
			wantErr:                "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_D4s_v2' is unavailable in region 'eastus'",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "worker and master skus are distinct, both invalid",
			workerProfile1Sku:      "Standard_E104i_v5",
			workerProfile2Sku:      "Standard_E104i_v5",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_L8s_v2",
			availableSku2:          "Standard_L16s_v2",
			wantErr:                "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_D4s_v2' is unavailable in region 'eastus'",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "unable to retrieve skus information",
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			resourceSkusClientErr:  errors.New("unable to retrieve skus information"),
			wantErr:                "unable to retrieve skus information",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "desired worker sku doesn't exist in the target region",
			workerProfile1Sku:      "Standard_L80",
			workerProfile2Sku:      "Standard_L80",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_D4s_v2",
			wantErr:                "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "desired master sku doesn't exist in the target region",
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_L80",
			availableSku:           "Standard_D4s_v2",
			wantErr:                "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:                   "one valid workerprofile and one invalid workerprofile",
			workerProfile1Sku:      "Standard_L80",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_D4s_v2",
			wantErr:                "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is unavailable in region 'eastus'",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
		},
		{
			name:         "worker sku exists in region but is not available in subscription",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			workerProfile1Sku:      "Standard_L80",
			workerProfile2Sku:      "Standard_L80",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_D4s_v2",
			restrictedSku:          "Standard_L80",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
			wantErr:                "400: InvalidParameter: properties.workerProfiles[0].VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
		{
			name:         "master sku exists in region but is not available in subscription",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_L80",
			availableSku:           "Standard_D4s_v2",
			restrictedSku:          "Standard_L80",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
			wantErr:                "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
		{
			name:         "sku is restricted in a single zone",
			restrictions: mgmtcompute.NotAvailableForSubscription,
			restrictionLocation: &[]string{
				"eastus",
			},
			restrictedZones:        []string{"3"},
			workerProfile1Sku:      "Standard_D4s_v2",
			workerProfile2Sku:      "Standard_D4s_v2",
			masterProfileSku:       "Standard_L80",
			availableSku:           "Standard_D4s_v2",
			restrictedSku:          "Standard_L80",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
			wantErr:                "400: InvalidParameter: properties.masterProfile.VMSize: The selected SKU 'Standard_L80' is restricted in region 'eastus' for selected subscription",
		},
		{
			name:                   "worker SKU does not have encryptionAtHost",
			workerProfile1Sku:      "Standard_E104i_v5",
			workerProfile2Sku:      "Standard_E104i_v5",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_E104i_v5",
			availableSku2:          "Standard_D4s_v2",
			masterEncryptionAtHost: api.EncryptionAtHostDisabled,
			workerEncryptionAtHost: api.EncryptionAtHostEnabled,
			wantErr:                "400: InvalidParameter: properties.workerProfiles[0].encryptionAtHost: The selected SKU 'Standard_E104i_v5' does not support encryption at host.",
		},
		{
			name:                   "master SKU does not have encryptionAtHost",
			workerProfile1Sku:      "Standard_E104i_v5",
			workerProfile2Sku:      "Standard_E104i_v5",
			masterProfileSku:       "Standard_D4s_v2",
			availableSku:           "Standard_E104i_v5",
			availableSku2:          "Standard_D4s_v2",
			masterEncryptionAtHost: api.EncryptionAtHostEnabled,
			workerEncryptionAtHost: api.EncryptionAtHostDisabled,
			wantErr:                "400: InvalidParameter: properties.masterProfile.encryptionAtHost: The selected SKU 'Standard_D4s_v2' does not support encryption at host.",
		},
		{
			name:                      "worker SKU has encryptionAtHost",
			workerProfile1Sku:         "Standard_E104i_v5",
			workerProfile2Sku:         "Standard_E104i_v5",
			masterProfileSku:          "Standard_D4s_v2",
			availableSku:              "Standard_E104i_v5",
			availableSku2:             "Standard_D4s_v2",
			masterEncryptionAtHost:    api.EncryptionAtHostDisabled,
			workerEncryptionAtHost:    api.EncryptionAtHostEnabled,
			availableSkuHasEncryption: true,
		},
		{
			name:                       "master SKU has encryptionAtHost",
			workerProfile1Sku:          "Standard_E104i_v5",
			workerProfile2Sku:          "Standard_E104i_v5",
			masterProfileSku:           "Standard_D4s_v2",
			availableSku:               "Standard_E104i_v5",
			availableSku2:              "Standard_D4s_v2",
			masterEncryptionAtHost:     api.EncryptionAtHostEnabled,
			workerEncryptionAtHost:     api.EncryptionAtHostDisabled,
			availableSku2HasEncryption: true,
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
							VMSize:           api.VMSize(tt.workerProfile1Sku),
							EncryptionAtHost: tt.workerEncryptionAtHost,
						},
						{
							VMSize:           api.VMSize(tt.workerProfile2Sku),
							EncryptionAtHost: tt.workerEncryptionAtHost,
						},
					},
					MasterProfile: api.MasterProfile{
						VMSize:           api.VMSize(tt.masterProfileSku),
						EncryptionAtHost: tt.masterEncryptionAtHost,
					},
				},
			}

			encryptionAtHost := func(enabled bool) *string {
				if enabled {
					return to.StringPtr("True")
				}
				return to.StringPtr("False")
			}

			skus := []mgmtcompute.ResourceSku{
				{
					Name:      &tt.availableSku,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &[]string{"1, 2, 3"}},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{
						{
							Name:  to.StringPtr("EncryptionAtHostSupported"),
							Value: encryptionAtHost(tt.availableSkuHasEncryption),
						},
					},
					ResourceType: to.StringPtr("virtualMachines"),
				},
				{
					Name:      &tt.availableSku2,
					Locations: &[]string{"eastus"},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &[]string{"1, 2, 3"}},
					},
					Restrictions: &[]mgmtcompute.ResourceSkuRestrictions{},
					Capabilities: &[]mgmtcompute.ResourceSkuCapabilities{
						{
							Name:  to.StringPtr("EncryptionAtHostSupported"),
							Value: encryptionAtHost(tt.availableSku2HasEncryption),
						},
					},
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

			if tt.wpStatus {
				oc.Properties.WorkerProfiles = nil
				oc.Properties.WorkerProfilesStatus = []api.WorkerProfile{
					{
						VMSize: api.VMSize(tt.workerProfile1Sku),
					},
					{
						VMSize: api.VMSize(tt.workerProfile2Sku),
					},
				}
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
