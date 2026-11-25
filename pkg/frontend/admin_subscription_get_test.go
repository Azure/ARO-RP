package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminGetSubscriptionDocument(t *testing.T) {
	ctx := t.Context()
	type test struct {
		name           string
		subscriptionID string
		fixture        func(f *testdatabase.Fixture)
		compareOption  cmp.Option
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
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: "00000000-0000-0000-0000-000000000001",
					Subscription: &api.Subscription{
						State:      api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{},
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: map[string]any{
				"properties": map[string]any{},
				"state":      "Registered",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			ti := newTestInfra(t).WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/subscriptions/"+tt.subscriptionID,
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			r.Equal(tt.wantStatusCode, resp.StatusCode)

			if tt.wantError != "" {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
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
