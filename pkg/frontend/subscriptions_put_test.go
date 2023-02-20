package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPutSubscription(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		request        func(*api.Subscription)
		fixture        func(*testdatabase.Fixture)
		dbError        error
		wantDbDoc      *api.SubscriptionDocument
		wantStatusCode int
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "add a new subscription - registered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateRegistered
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - warned state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateWarned
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateWarned,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - suspended state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateSuspended
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateSuspended,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - unregistered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateUnregistered
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateUnregistered,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - deleted state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateDeleted
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID:       mockSubID,
				Deleting: true,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateDeleted,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - request contains pii",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateRegistered
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed", AccountOwner: &api.AccountOwnerProfile{Email: "email@example.com"}}
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{TenantID: "changed", AccountOwner: &api.AccountOwnerProfile{Email: ""}},
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "update an existing subscription - registered",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateWarned
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateWarned,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - warned state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateSuspended
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateWarned,
					},
				})
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateSuspended,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - suspended state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateDeleted
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateSuspended,
					},
				})
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID:       mockSubID,
				Deleting: true,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateDeleted,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - unregistered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateRegistered
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateUnregistered,
					},
				})
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - deleted state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateUnregistered
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID:       mockSubID,
					Deleting: true,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateDeleted,
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidSubscriptionState: : Request is not allowed in subscription in state 'Deleted'.`,
		},
		{
			name:           "internal error",
			dbError:        errors.New("random error"),
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftClusters()
			defer ti.done()

			if tt.dbError != nil {
				ti.subscriptionsClient.SetError(tt.dbError)
				ti.openShiftClustersClient.SetError(tt.dbError)
			}

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			sub := &api.Subscription{}
			if tt.request != nil {
				tt.request(sub)
			}

			resp, b, err := ti.request(http.MethodPut,
				fmt.Sprintf("https://server/subscriptions/%s?api-version=2.0", mockSubID),
				http.Header{
					"Content-Type": []string{"application/json"},
				}, sub)
			if err != nil {
				t.Error(err)
			}

			var wantResponse interface{}
			if tt.wantDbDoc != nil {
				wantResponse = tt.wantDbDoc.Subscription
				ti.checker.AddSubscriptionDocuments(tt.wantDbDoc)
				errs := ti.checker.CheckSubscriptions(ti.subscriptionsClient)
				for _, i := range errs {
					t.Error(i)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
