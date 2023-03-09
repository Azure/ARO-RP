package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAppLensDetectors(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		detectorID     string
		getDetector    bool
		mocks          func(*test, *mock_adminactions.MockAzureActions)
		method         string
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			method:      http.MethodGet,
			name:        "list applens detectors",
			resourceID:  fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			detectorID:  "",
			getDetector: false,
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					AppLensListDetectors(gomock.Any()).
					Return([]byte(`{"Kind": "test"}`), nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"Kind": "test"}` + "\n"),
		},
		{
			method:      http.MethodGet,
			name:        "get applens detector",
			resourceID:  fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			detectorID:  "testdetector",
			getDetector: true,
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().
					AppLensGetDetector(gomock.Any(), tt.detectorID).
					Return([]byte(`{"Kind": "test"}`), nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"Kind": "test"}` + "\n"),
		},
		{
			method:      http.MethodGet,
			name:        "no applens detector specified",
			resourceID:  fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			detectorID:  "",
			getDetector: true,
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: NotFound: : The requested path could not be found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.mocks(tt, a)

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			finalURL := fmt.Sprintf("https://server%s/detectors", tt.resourceID)
			if tt.getDetector {
				finalURL = fmt.Sprintf("%s/%s", finalURL, tt.detectorID)
			}

			resp, b, err := ti.request(http.MethodGet, finalURL, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
