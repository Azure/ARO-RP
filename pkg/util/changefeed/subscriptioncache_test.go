package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestSubscriptionChangefeed(t *testing.T) {
	testCases := []struct {
		desc      string
		validOnly bool
		expected  map[string]subscriptionInfo
	}{
		{
			desc:      "only valid subscriptions",
			validOnly: true,
			expected: map[string]subscriptionInfo{
				"9187ef95-a9cc-487d-80df-f85e615cf926": {
					State: api.SubscriptionStateRegistered, TenantID: "41441389-d1c2-4ade-b95d-99445d169804",
				},
				"fb4b6d1a-5ede-4ed4-8a37-96b9c5397616": {
					State: api.SubscriptionStateRegistered, TenantID: "84ca9ee8-04fd-4309-9968-62d22696192c",
				},
				// created after the initial feeding
				"07e31457-5d73-4a99-a316-52a226179267": {
					State: api.SubscriptionStateRegistered, TenantID: "6baab395-b792-4ee1-99e7-4f8315be1543",
				},
			},
		},
		{
			desc:      "all (non-deleted) subscriptions",
			validOnly: false,
			expected: map[string]subscriptionInfo{
				"9187ef95-a9cc-487d-80df-f85e615cf926": {
					State: api.SubscriptionStateRegistered, TenantID: "41441389-d1c2-4ade-b95d-99445d169804",
				},
				"fb4b6d1a-5ede-4ed4-8a37-96b9c5397616": {
					State: api.SubscriptionStateRegistered, TenantID: "84ca9ee8-04fd-4309-9968-62d22696192c",
				},
				"ea93be31-c21d-424b-ac04-fcb6f20804dc": {
					State: api.SubscriptionStateSuspended, TenantID: "6b456b5d-34c0-4ba1-b80f-5e38032a9003",
				},
				// created after the initial feeding
				"07e31457-5d73-4a99-a316-52a226179267": {
					State: api.SubscriptionStateRegistered, TenantID: "6baab395-b792-4ee1-99e7-4f8315be1543",
				},
				"f9664b2f-0ea1-4401-be48-f7611f58c295": {
					State: api.SubscriptionStateWarned, TenantID: "bb0ba6ad-abb8-4b81-8c65-4081bfed7928",
				},
				"8c90b62a-3783-4ea6-a8c8-cbaee4667ffd": {
					State: api.SubscriptionStateSuspended, TenantID: "21e4577c-464b-4435-b9db-491274406c26",
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			startedTime := time.Now().UnixNano()
			subscriptionsDB, subscriptionsClient := testdatabase.NewFakeSubscriptions()
			_, log := testlog.LogForTesting(t)

			// need to register the changefeed before making documents
			subscriptionChangefeed := subscriptionsDB.ChangeFeed()

			fixtures := testdatabase.NewFixture().WithSubscriptions(subscriptionsDB)

			fixtures.AddSubscriptionDocuments(
				&api.SubscriptionDocument{
					ID: "9187ef95-a9cc-487d-80df-f85e615cf926",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "41441389-d1c2-4ade-b95d-99445d169804",
						},
					},
				},
				&api.SubscriptionDocument{
					ID: "fb4b6d1a-5ede-4ed4-8a37-96b9c5397616",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "84ca9ee8-04fd-4309-9968-62d22696192c",
						},
					},
				},
				&api.SubscriptionDocument{
					// Will be set to suspended later
					ID: "8c90b62a-3783-4ea6-a8c8-cbaee4667ffd",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "21e4577c-464b-4435-b9db-491274406c26",
						},
					},
				},
				&api.SubscriptionDocument{
					// Will be set to deleted later
					ID: "4e07b0f5-c789-4817-9079-94012b04e1c9",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "8b0c5075-888e-4df6-b0f4-942dad50f132",
						},
					},
				},
				&api.SubscriptionDocument{
					ID: "ea93be31-c21d-424b-ac04-fcb6f20804dc",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateSuspended,
						Properties: &api.SubscriptionProperties{
							TenantID: "6b456b5d-34c0-4ba1-b80f-5e38032a9003",
						},
					},
				},
				// Deleted will not show up in the cache
				&api.SubscriptionDocument{
					ID: "d95482a5-2cb5-4397-9e85-29c2f828ae5d",
					Subscription: &api.Subscription{
						State: api.SubscriptionStateDeleted,
						Properties: &api.SubscriptionProperties{
							TenantID: "e193e232-5be1-4208-8c32-2a2ab30c0b13",
						},
					},
				},
			)
			require.NoError(t, fixtures.Create())

			cache := NewSubscriptionsChangefeedCache(tC.validOnly)

			stop := make(chan struct{})
			defer close(stop)

			go RunChangefeed(t.Context(), log, subscriptionChangefeed, 100*time.Microsecond, 1, cache, stop)

			cache.WaitForInitialPopulation()
			assert.Eventually(t, subscriptionsClient.AllIteratorsConsumed, time.Second, 1*time.Millisecond)

			// Create some after initially populated
			_, err := subscriptionsDB.Create(t.Context(), &api.SubscriptionDocument{
				ID: "07e31457-5d73-4a99-a316-52a226179267",
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: "6baab395-b792-4ee1-99e7-4f8315be1543",
					},
				},
			})
			require.NoError(t, err)
			_, err = subscriptionsDB.Create(t.Context(), &api.SubscriptionDocument{
				ID: "f9664b2f-0ea1-4401-be48-f7611f58c295",
				Subscription: &api.Subscription{
					State: api.SubscriptionStateWarned,
					Properties: &api.SubscriptionProperties{
						TenantID: "bb0ba6ad-abb8-4b81-8c65-4081bfed7928",
					},
				},
			})
			require.NoError(t, err)

			// Update one that is already populated
			old, err := subscriptionsDB.Get(t.Context(), "9187ef95-a9cc-487d-80df-f85e615cf926")
			require.NoError(t, err)
			_, err = subscriptionsDB.Update(t.Context(), &api.SubscriptionDocument{
				ID:   "9187ef95-a9cc-487d-80df-f85e615cf926",
				ETag: old.ETag,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: "41441389-d1c2-4ade-b95d-99445d169804",
					},
				},
			})
			require.NoError(t, err)
			assert.Eventually(t, subscriptionsClient.AllIteratorsConsumed, time.Second, 1*time.Millisecond)

			// Switch a registered to suspended
			old2, err := subscriptionsDB.Get(t.Context(), "8c90b62a-3783-4ea6-a8c8-cbaee4667ffd")
			require.NoError(t, err)
			_, err = subscriptionsDB.Update(t.Context(), &api.SubscriptionDocument{
				ID:   "8c90b62a-3783-4ea6-a8c8-cbaee4667ffd",
				ETag: old2.ETag,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateSuspended,
					Properties: &api.SubscriptionProperties{
						TenantID: "21e4577c-464b-4435-b9db-491274406c26",
					},
				},
			})
			require.NoError(t, err)
			assert.Eventually(t, subscriptionsClient.AllIteratorsConsumed, time.Second, 1*time.Millisecond)

			// Switch a registered to deleted
			old3, err := subscriptionsDB.Get(t.Context(), "4e07b0f5-c789-4817-9079-94012b04e1c9")
			require.NoError(t, err)
			_, err = subscriptionsDB.Update(t.Context(), &api.SubscriptionDocument{
				ID:   "4e07b0f5-c789-4817-9079-94012b04e1c9",
				ETag: old3.ETag,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateDeleted,
					Properties: &api.SubscriptionProperties{
						TenantID: "8b0c5075-888e-4df6-b0f4-942dad50f132",
					},
				},
			})
			require.NoError(t, err)
			assert.Eventually(t, subscriptionsClient.AllIteratorsConsumed, time.Second, 1*time.Millisecond)

			// Validate the expected cache contents
			assert.Equal(t, tC.expected, maps.Collect(cache.subs.All()))

			// Validate we can get one of the subscriptions
			sub, ok := cache.GetSubscription("9187ef95-a9cc-487d-80df-f85e615cf926")
			assert.True(t, ok, "fetching sub from cache")
			assert.Equal(t, subscriptionInfo{
				State: api.SubscriptionStateRegistered, TenantID: "41441389-d1c2-4ade-b95d-99445d169804",
			}, sub)

			// Validate the cache size is correct
			assert.Equal(t, len(tC.expected), cache.GetCacheSize())

			// Validate the last processed is set and is past now
			last, ok := cache.GetLastProcessed()
			assert.True(t, ok, "fetching last processed time")
			assert.Greater(t, last.UnixNano(), startedTime)
		})
	}
}

func TestSubscriptionChangefeedError(t *testing.T) {
	subscriptionsDB, subscriptionsClient := testdatabase.NewFakeSubscriptions()
	hook, log := testlog.LogForTesting(t)

	subscriptionsClient.SetError(errors.New("oh no"))

	// need to register the changefeed before making documents
	subscriptionChangefeed := subscriptionsDB.ChangeFeed()

	cache := NewSubscriptionsChangefeedCache(true)

	stop := make(chan struct{})
	defer close(stop)

	// set it on a massive loop so it only runs once
	go RunChangefeed(t.Context(), log, subscriptionChangefeed, 10000*time.Hour, 1, cache, stop)

	// it'll print the log when on the first loop, use Eventually so that we're
	// not in a race with the goroutine we spawned
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		require.NoError(collect, testlog.AssertLoggingOutput(hook, []testlog.ExpectedLogEntry{
			{
				"level": gomega.Equal(logrus.ErrorLevel),
				"msg":   gomega.Equal("while calling iterator.Next(): oh no"),
			},
		}))
	}, time.Second, time.Millisecond)

	// Empty cache
	assert.Equal(t, map[string]subscriptionInfo{}, maps.Collect(cache.subs.All()))
}
