package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminDeleteManagedResource(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

	ctx := context.Background()

	type test struct {
		name              string
		resourceID        string
		managedResourceID string
		mocks             func(*test, *mock_adminactions.MockAzureActions)
		wantStatusCode    int
		wantResponse      []byte
		wantError         string
	}

	for _, tt := range []*test{
		{
			name:              "delete managed resource within cluster managed resourcegroup",
			resourceID:        testdatabase.GetResourcePath(mockSubID, "resourceName"),
			managedResourceID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceDeleteAndWait(gomock.Any(), tt.managedResourceID).Return(nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:              "delete managed resource not within cluster managed resourcegroup fails",
			resourceID:        testdatabase.GetResourcePath(mockSubID, "resourceName"),
			managedResourceID: fmt.Sprintf("/subscriptions/%s/resourceGroups/notmanagedresourcegroup/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The resource /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/notmanagedresourcegroup/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083 is not within the cluster's managed resource group /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster.",
		},
		{
			name:              "delete a resource that doesn't exist fails",
			resourceID:        testdatabase.GetResourcePath(mockSubID, "resourceName"),
			managedResourceID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceDeleteAndWait(gomock.Any(), tt.managedResourceID).Return(autorest.DetailedError{StatusCode: 404})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      fmt.Sprintf("404: NotFound: : The resource '%s' could not be found.", fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083", mockSubID)),
		},
		{
			name:              "cannot delete resources in the deny list",
			resourceID:        testdatabase.GetResourcePath(mockSubID, "resourceName"),
			managedResourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/test-cluster/providers/Microsoft.Network/privateLinkServices/infraID", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().ResourceDeleteAndWait(gomock.Any(), tt.managedResourceID).Return(api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
					fmt.Sprintf("deletion of resource /subscriptions/%s/resourcegroups/test-cluster/providers/Microsoft.Network/privateLinkServices/infraID is forbidden", mockSubID)),
				)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
				fmt.Sprintf("deletion of resource /subscriptions/%s/resourcegroups/test-cluster/providers/Microsoft.Network/privateLinkServices/infraID is forbidden", mockSubID)).Error(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()
			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
						},
					},
				},
			})
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.mocks(tt, a)

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)
			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/deletemanagedresource?managedResourceID=%s", tt.resourceID, tt.managedResourceID),
				nil, nil)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
