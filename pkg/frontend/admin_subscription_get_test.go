package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminGetSubscriptionDocument(t *testing.T) {
	type test struct {
		name           string
		subscriptionID string
		fixture        func(f *testdatabase.Fixture)
		wantStatusCode int
		wantError      string
		wantResponse   map[string]any
	}

	for _, tt := range []*test{
		{
			name:           "no subscription document found",
			subscriptionID: "non-existent-id",
			wantError:      "404: NotFound: : subscription not found",
			fixture:        func(f *testdatabase.Fixture) {},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "get subscription document",
			subscriptionID: "00000000-0000-0000-0000-000000000001",
			fixture: func(f *testdatabase.Fixture) {
				doc := &api.SubscriptionDocument{
					ID: "00000000-0000-0000-0000-000000000001",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "1234",
						},
						Overrides: &api.SubscriptionOverrides{
							InstallationZones: []string{"1"},
						},
					},
				}
				doc.Subscription.Properties.CodecMissingField([]byte("otherdata"), "data")
				f.AddSubscriptionDocuments(doc)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]any{
				"properties": map[string]any{
					// data held in MissingFields
					"otherdata": "data",
					"tenantId":  "1234",
				},
				"overrides": map[string]any{
					"installationZones": []any{"1"},
				},
				"state": "Registered",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			ti := newTestInfra(t).WithSubscriptions()
			defer ti.done()

			r.NoError(ti.buildFixtures(tt.fixture))

			f, err := NewFrontend(t.Context(), ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			r.NoError(err)

			ti.Run(f)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/subscriptions/"+tt.subscriptionID,
				nil, nil)
			r.NoError(err)

			r.Equal(tt.wantStatusCode, resp.StatusCode)

			if tt.wantError != "" {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				r.NoError(err)
				r.Equal(tt.wantError, cloudErr.Error())
				return
			}

			res := map[string]any{}
			err = json.Unmarshal(b, &res)
			r.NoError(err)
			r.Equal(tt.wantResponse, res)
		})
	}
}
