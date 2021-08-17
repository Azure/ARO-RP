package dynamic

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
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
)

func TestValidateVMSku(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		restrictions          mgmtcompute.ResourceSkuRestrictionsReasonCode
		restrictionLocation   *[]string
		targetLocation        string
		workerProfile1Sku     string
		workerProfile2Sku     string
		masterProfileSku      string
		availableSku          string
		restrictedSku         string
		resourceSkusClientErr error
		wantErr               string
	}{
		{
			name:              "worker and master sku are valid",
			workerProfile1Sku: "Standard_D4s_v2",
			workerProfile2Sku: "Standard_D4s_v2",
			masterProfileSku:  "Standard_D4s_v2",
			availableSku:      "Standard_D4s_v2",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.targetLocation == "" {
				tt.targetLocation = "eastus"
			}

			controller := gomock.NewController(t)
			defer controller.Finish()

			oc := &api.OpenShiftCluster{
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
					Name:      &tt.restrictedSku,
					Locations: &[]string{tt.targetLocation},
					LocationInfo: &[]mgmtcompute.ResourceSkuLocationInfo{
						{Zones: &[]string{"1, 2, 3"}},
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
				List(gomock.Any(), fmt.Sprintf("location eq %v", tt.targetLocation)).
				Return(skus, tt.resourceSkusClientErr)

			dv := dynamic{
				authorizerType:     AuthorizerClusterServicePrincipal,
				log:                logrus.NewEntry(logrus.StandardLogger()),
				resourceSkusClient: resourceSkusClient,
			}

			err := dv.ValidateVMSku(context.Background(), tt.targetLocation, subscriptionID, oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
