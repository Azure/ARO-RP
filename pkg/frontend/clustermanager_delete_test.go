package frontend

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestDeleteClusterManagerConfiguration(t *testing.T) {
	ctx := context.Background()

	mockSubscriptionId := "00000000-0000-0000-0000-000000000000"
	resourcePayload := "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAic2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo="

	type test struct {
		name            string
		ocmResourceType string
		ocmResourceName string
		clusterName     string
		apiVersion      string
		fixture         func(*testdatabase.Fixture, *test, string)
		wantStatusCode  int
		wantError       string
	}
	createSingleDocument := func(f *testdatabase.Fixture, tt *test, resourceKey string) {
		f.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubscriptionId,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: "11111111-1111-1111-1111-111111111111",
				},
			},
		})
		f.AddClusterManagerConfigurationDocuments(
			&api.ClusterManagerConfigurationDocument{
				ID:  mockSubscriptionId,
				Key: resourceKey,
				ClusterManagerConfiguration: &api.ClusterManagerConfiguration{
					Name: tt.ocmResourceName,
					Properties: api.ClusterManagerConfigurationProperties{
						Resources: []byte(resourcePayload),
					},
				},
			},
		)
	}

	for _, tt := range []*test{
		{
			name:            "single syncset",
			ocmResourceType: "syncSet",
			ocmResourceName: "deleteSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			wantStatusCode:  http.StatusOK,
		},
		{
			name:            "does not exist",
			ocmResourceType: "syncSet",
			ocmResourceName: "deleteSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture: func(f *testdatabase.Fixture, tt *test, resourceKey string) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubscriptionId,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename/syncset/deletesyncset' under resource group 'resourcegroup' was not found.",
		},
		{
			name:            "wrong version",
			ocmResourceType: "syncSet",
			ocmResourceName: "deleteSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-04-01",
			fixture:         createSingleDocument,
			wantStatusCode:  http.StatusBadRequest,
			wantError:       "400: InvalidResourceType: : The resource type 'openshiftclusters' could not be found in the namespace 'microsoft.redhatopenshift' for api version '2022-04-01'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithClusterManagerConfigurations().WithSubscriptions()
			defer ti.done()

			resourceKey := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/%s/%s",
				mockSubscriptionId,
				tt.ocmResourceType,
				tt.ocmResourceName)

			err := ti.buildFixtures(func(f *testdatabase.Fixture) { tt.fixture(f, tt, resourceKey) })
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, ti.clusterManagerDatabase, nil, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodDelete,
				fmt.Sprintf("https://server%s?api-version=%s",
					resourceKey,
					tt.apiVersion,
				),
				nil, nil)
			if err != nil {
				t.Fatalf("%s: %s", err, string(b))
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, "")
			if err != nil {
				t.Errorf("%s: %s", err, string(b))
			}
		})
	}
}
