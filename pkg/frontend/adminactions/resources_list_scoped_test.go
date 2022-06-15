package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestResourcesListScoped(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name         string
		resourceType string
		mocks        func(resources *mock_features.MockResourcesClient, resourceType string)
		wantResponse []byte
		wantError    string
	}

	for _, tt := range []*test{
		{
			name:         "basic coverage",
			resourceType: "Microsoft.Compute/virtualNetworks",
			mocks: func(resources *mock_features.MockResourcesClient, resourceType string) {
				resources.EXPECT().ListByResourceGroup(gomock.Any(), "test-cluster", fmt.Sprintf("resourceType eq '%s'", resourceType), "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name:     to.StringPtr("vnet-1"),
						ID:       to.StringPtr("/subscriptions/id"),
						Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
						Location: to.StringPtr("eastus"),
					},
				}, nil)
			},
			wantResponse: []byte(`[{"id":"/subscriptions/id","name":"vnet-1","type":"Microsoft.Network/virtualNetworks","location":"eastus"}]`),
		},
		{
			name:         "error getting the resources from ARM",
			resourceType: "Microsoft.Compute/virtualMachines",
			mocks: func(resources *mock_features.MockResourcesClient, resourceType string) {
				resources.EXPECT().ListByResourceGroup(gomock.Any(), "test-cluster", fmt.Sprintf("resourceType eq '%s'", resourceType), "", nil).Return([]mgmtfeatures.GenericResourceExpanded{}, fmt.Errorf("couldn't get resources."))
			},
			wantError: "couldn't get resources.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return("eastus")

			resources := mock_features.NewMockResourcesClient(controller)

			tt.mocks(resources, tt.resourceType)

			a := azureActions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
						},
					},
				},

				resources: resources,
			}

			b, err := a.ResourcesListScoped(ctx, tt.resourceType)

			if tt.wantError == "" {
				if tt.wantResponse != nil {
					if !bytes.Equal(b, tt.wantResponse) {
						t.Errorf("Wanted %s, got %s", tt.wantResponse, string(b))
					}
				}
			} else {
				if err.Error() != tt.wantError {
					t.Fatal(err)
				}
			}
		})
	}
}
