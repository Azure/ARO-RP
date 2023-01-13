package frontend

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestPutOrPatchClusterManagerConfiguration(t *testing.T) {
	ctx := context.Background()

	mockSubscriptionId := "00000000-0000-0000-0000-000000000000"
	tenantId := "11111111-1111-1111-1111-111111111111"
	resourcePayload := "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAic2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo="
	modifiedPayload := "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAibW9kaWZpZWQtc2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo="

	type test struct {
		name            string
		ocmResourceType string
		ocmResourceName string
		clusterName     string
		apiVersion      string
		fixture         func(*testdatabase.Fixture, *test, string)
		requestMethod   string
		requestBody     string
		wantStatusCode  int
		wantResponse    *v20220904.SyncSet
		wantError       string
	}
	createSingleDocument := func(f *testdatabase.Fixture, tt *test, resourceKey string) {
		f.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubscriptionId,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: tenantId,
				},
			},
		})
		f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
			Key: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionId, "resourceName")),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: testdatabase.GetResourcePath(mockSubscriptionId, "resourceName"),
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s", mockSubscriptionId, "rg01"),
					},
				},
			},
		})
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
	noDocuments := func(f *testdatabase.Fixture, tt *test, resourceKey string) {
		f.AddSubscriptionDocuments(&api.SubscriptionDocument{
			ID: mockSubscriptionId,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: tenantId,
				},
			},
		})
		f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
			Key: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionId, "resourceName")),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: testdatabase.GetResourcePath(mockSubscriptionId, "resourceName"),
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s", mockSubscriptionId, tt.clusterName),
					},
				},
			},
		})
	}

	for _, tt := range []*test{
		{
			name:            "single syncset - put",
			ocmResourceType: "syncSet",
			ocmResourceName: "putSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			requestMethod:   http.MethodPut,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusOK,
			wantResponse: &v20220904.SyncSet{
				Name: "putSyncSet",
				Properties: v20220904.SyncSetProperties{
					Resources: modifiedPayload,
				},
			},
		},
		{
			name:            "single syncset - patch",
			ocmResourceType: "syncSet",
			ocmResourceName: "patchSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			requestMethod:   http.MethodPatch,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusOK,
			wantResponse: &v20220904.SyncSet{
				Name: "patchSyncSet",
				Properties: v20220904.SyncSetProperties{
					Resources: modifiedPayload,
				},
			},
		},
		{
			name:            "single syncset - put create",
			ocmResourceType: "syncSet",
			ocmResourceName: "putNewSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         noDocuments,
			requestMethod:   http.MethodPut,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusOK,
			wantResponse: &v20220904.SyncSet{
				Name: "putnewsyncset",
				Type: "Microsoft.RedHatOpenShift/SyncSet",
				ID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/syncSet/putNewSyncSet",
				Properties: v20220904.SyncSetProperties{
					Resources: modifiedPayload,
				},
			},
		},
		{
			name:            "patching nonexistent syncset",
			ocmResourceType: "syncSet",
			ocmResourceName: "patchNewSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         noDocuments,
			requestMethod:   http.MethodPatch,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusNotFound,
			wantError:       "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename/syncset/patchnewsyncset' under resource group 'resourcegroup' was not found.",
		},
		{
			name:            "unsupported api version",
			ocmResourceType: "syncset",
			ocmResourceName: "patchSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-04-01",
			fixture:         createSingleDocument,
			requestMethod:   http.MethodPatch,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusBadRequest,
			wantError:       "400: InvalidResourceType: : the resource type 'syncset' is not valid for api version '2022-04-01'",
		},
		{
			name:            "unsupported resource type",
			ocmResourceType: "unsupported",
			ocmResourceName: "patchSyncSet",
			clusterName:     "myCluster",
			apiVersion:      "2022-09-04",
			fixture:         createSingleDocument,
			requestMethod:   http.MethodPatch,
			requestBody:     modifiedPayload,
			wantStatusCode:  http.StatusBadRequest,
			wantError:       "400: InvalidResourceType: : the resource type 'unsupported' is not valid for api version '2022-09-04'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfraWithFeatures(t, map[env.Feature]bool{env.FeatureRequireD2sV3Workers: false, env.FeatureDisableReadinessDelay: false, env.FeatureEnableOCMEndpoints: true}).WithClusterManagerConfigurations().WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			resourceKey := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/%s/%s",
				mockSubscriptionId,
				tt.ocmResourceType,
				tt.ocmResourceName)

			err := ti.buildFixtures(func(f *testdatabase.Fixture) { tt.fixture(f, tt, resourceKey) })
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(tt.requestMethod,
				fmt.Sprintf("https://server%s?api-version=%s",
					resourceKey,
					tt.apiVersion,
				),
				http.Header{
					"Content-Type": []string{"application/json"},
				}, tt.requestBody)
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

var populatedSystemData = api.SystemData{
	LastModifiedAt:     &time.Time{},
	LastModifiedBy:     "test-user",
	LastModifiedByType: api.CreatedByTypeUser,
	CreatedAt:          &time.Time{},
	CreatedBy:          "test-user",
	CreatedByType:      api.CreatedByTypeUser,
}

func TestEnrichSyncSetSystemData(t *testing.T) {
	doc := api.ClusterManagerConfigurationDocument{
		SyncSet: &api.SyncSet{},
	}

	enrichSyncSetSystemData(&doc, &populatedSystemData)

	if err := compareSystemData(*doc.SyncSet.SystemData, populatedSystemData); err != nil {
		t.Fatal(err)
	}
}

func TestEnrichSyncIdentityProviderSystemData(t *testing.T) {
	doc := api.ClusterManagerConfigurationDocument{
		SyncIdentityProvider: &api.SyncIdentityProvider{},
	}

	enrichSyncIdentityProviderSystemData(&doc, &populatedSystemData)

	if err := compareSystemData(*doc.SyncIdentityProvider.SystemData, populatedSystemData); err != nil {
		t.Fatal(err)
	}
}

func TestEnrichMachinePoolSystemData(t *testing.T) {
	doc := api.ClusterManagerConfigurationDocument{
		MachinePool: &api.MachinePool{},
	}

	enrichMachinePoolSystemData(&doc, &populatedSystemData)

	if err := compareSystemData(*doc.MachinePool.SystemData, populatedSystemData); err != nil {
		t.Fatal(err)
	}
}

func TestEnrichSecretSystemData(t *testing.T) {
	doc := api.ClusterManagerConfigurationDocument{
		Secret: &api.Secret{},
	}

	enrichSecretSystemData(&doc, &populatedSystemData)

	if err := compareSystemData(*doc.Secret.SystemData, populatedSystemData); err != nil {
		t.Fatal(err)
	}
}

func compareSystemData(docSystemData, expectedSystemData api.SystemData) error {
	if docSystemData.CreatedAt == nil || docSystemData.CreatedAt != expectedSystemData.CreatedAt {
		return fmt.Errorf("CreatedAt was %q expected %q", docSystemData.CreatedAt, expectedSystemData.CreatedAt)
	}
	if docSystemData.CreatedBy == "" || docSystemData.CreatedBy != expectedSystemData.CreatedBy {
		return fmt.Errorf("CreatedBy was %q expected %q", docSystemData.CreatedBy, expectedSystemData.CreatedBy)
	}
	if docSystemData.CreatedByType == "" || docSystemData.CreatedByType != expectedSystemData.CreatedByType {
		return fmt.Errorf("CreatedByType was %q expected %q", docSystemData.CreatedByType, expectedSystemData.CreatedByType)
	}
	if docSystemData.LastModifiedAt == nil || docSystemData.LastModifiedAt != expectedSystemData.LastModifiedAt {
		return fmt.Errorf("LastModifiedAt was %q expected %q", docSystemData.LastModifiedAt, expectedSystemData.LastModifiedAt)
	}
	if docSystemData.LastModifiedBy == "" || docSystemData.LastModifiedBy != expectedSystemData.LastModifiedBy {
		return fmt.Errorf("LastModifiedBy was %q expected %q", docSystemData.LastModifiedBy, expectedSystemData.LastModifiedBy)
	}
	if docSystemData.LastModifiedByType == "" || docSystemData.LastModifiedByType != expectedSystemData.LastModifiedByType {
		return fmt.Errorf("LastModifiedByType was %q expected %q", docSystemData.LastModifiedByType, expectedSystemData.LastModifiedByType)
	}
	return nil
}
