package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func getQueuedSubscriptionDocuments(client cosmosdb.SubscriptionDocumentClient) (results []*api.SubscriptionDocument) {
	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	for _, r := range input.SubscriptionDocuments {
		if r.Deleting && int64(r.LeaseExpires) < time.Now().Unix() {
			results = append(results, r)
		}
	}
	return
}

func fakeSubscriptionsDequeueQuery(client cosmosdb.SubscriptionDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.SubscriptionDocumentRawIterator {
	docs := getQueuedSubscriptionDocuments(client)
	return cosmosdb.NewFakeSubscriptionDocumentIterator(docs, 0)
}

func fakeBillingRenewLeaseTrigger(ctx context.Context, doc *api.SubscriptionDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 60
	return nil
}

func fakeBillingRetryLaterTrigger(ctx context.Context, doc *api.SubscriptionDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 600
	return nil
}

func injectSubscriptions(c *cosmosdb.FakeSubscriptionDocumentClient) {
	c.SetQueryHandler(database.SubscriptionsDequeueQuery, fakeSubscriptionsDequeueQuery)

	c.SetTriggerHandler("renewLease", fakeBillingRenewLeaseTrigger)
	c.SetTriggerHandler("retryLater", fakeBillingRetryLaterTrigger)

	c.SetSorter(func(in []*api.SubscriptionDocument) {
		slices.SortFunc(in, func(a, b *api.SubscriptionDocument) int { return CompareIDable(a, b) })
	})
}
