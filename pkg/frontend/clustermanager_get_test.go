package frontend

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestGetClusterManagerConfiguration(t *testing.T) {
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
		wantResponse    *v20220904.SyncSet
		wantError       string
	}
	createSingleDocument := func(f *testdatabase.Fixture, tt *test, resourceKey string) {
		f.AddClusterManagerConfigurationDocuments(
			&api.ClusterManagerConfigurationDocument{
				ID:  mockSubscriptionId,
				Key: resourceKey,
				SyncSet: &api.SyncSet{
					Name: tt.ocmResourceName,
					Properties: api.SyncSetProperties{
						Resources: resourcePayload,
					},
				},
			},
		)
	}

	for _, tt := range []*test{
		{
			name:            "single syncset",
			ocmResourceType: "syncSet",
			ocmResourceName: "mySyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			wantStatusCode:  http.StatusOK,
			wantResponse: &v20220904.SyncSet{
				Name: "mySyncSet",
				Properties: v20220904.SyncSetProperties{
					Resources: resourcePayload,
				},
			},
		},
		{
			name:            "syncset is deleting",
			ocmResourceType: "syncSet",
			ocmResourceName: "myDeletingSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture: func(f *testdatabase.Fixture, tt *test, resourceKey string) {
				f.AddClusterManagerConfigurationDocuments(
					&api.ClusterManagerConfigurationDocument{
						ID:       mockSubscriptionId,
						Key:      resourceKey,
						Deleting: true,
						SyncSet: &api.SyncSet{
							Name: tt.ocmResourceName,
							Properties: api.SyncSetProperties{
								Resources: resourcePayload,
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on a resource marked for deletion.",
		},
		{
			name:            "wrong version",
			ocmResourceType: "syncset",
			ocmResourceName: "mySyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-04-01",
			fixture:         createSingleDocument,
			wantStatusCode:  http.StatusBadRequest,
			wantError:       "400: InvalidResourceType: : the resource type 'syncset' is not valid for api version '2022-04-01'",
		},
		{
			name:            "unsupported resource type",
			ocmResourceType: "unsupported",
			ocmResourceName: "invalidResourceType",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			wantStatusCode:  http.StatusBadRequest,
			wantError:       "400: InvalidResourceType: : the resource type 'unsupported' is not valid for api version '2022-09-04'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfraWithFeatures(t, map[env.Feature]bool{env.FeatureRequireD2sV3Workers: false, env.FeatureDisableReadinessDelay: false, env.FeatureEnableOCMEndpoints: true}).WithClusterManagerConfigurations()
			defer ti.done()

			resourceKey := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/%s/%s",
				mockSubscriptionId,
				tt.ocmResourceType,
				tt.ocmResourceName)

			err := ti.buildFixtures(func(f *testdatabase.Fixture) { tt.fixture(f, tt, resourceKey) })
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, ti.clusterManagerDatabase, nil, nil, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server%s?api-version=%s",
					resourceKey,
					tt.apiVersion,
				),
				nil, nil)
			if err != nil {
				t.Fatalf("%s: %s", err, string(b))
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Errorf("%s: %s", err, string(b))
			}
		})
	}
}
