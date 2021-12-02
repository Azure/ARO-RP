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

func TestOpenShiftNodesTop(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name               string
		resourceID         string
		resourceIDOverride string
		mocks              func(*test, *mock_adminactions.MockMetricActions)
		method             string
		wantStatusCode     int
		wantResponse       []byte
		wantError          string
	}

	for _, tt := range []*test{
		{
			method:     http.MethodGet,
			name:       "cluster exists",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, m *mock_adminactions.MockMetricActions) {
				m.EXPECT().
					TopNodes(gomock.Any()).
					Return([]byte(`{"Kind": "test"}`), nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []byte(`{"Kind": "test"}` + "\n"),
		},
		{
			method:             http.MethodGet,
			name:               "cluster does not exist",
			resourceID:         fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			resourceIDOverride: "/subscriptions/11100000-0000-0000-0000-000000000111/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName",
			mocks: func(tt *test, m *mock_adminactions.MockMetricActions) {
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", tt.method, tt.name), func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			m := mock_adminactions.NewMockMetricActions(ti.controller)
			tt.mocks(tt, m)

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

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, api.APIs, &noop.Noop{}, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.MetricActions, error) {
				return m, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resourceID := tt.resourceID
			if tt.resourceIDOverride != "" {
				resourceID = tt.resourceIDOverride
			}

			resp, b, err := ti.request(tt.method,
				fmt.Sprintf("https://server%s/topnodes?api-version=2020-04-30", resourceID),
				nil, nil)
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
