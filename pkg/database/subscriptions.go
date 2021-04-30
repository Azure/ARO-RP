package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

const SubscriptionsDequeueQuery string = `SELECT * FROM Subscriptions doc WHERE (doc.deleting ?? false) AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`

type subscriptions struct {
	c    cosmosdb.SubscriptionDocumentClient
	uuid string
}

// Subscriptions is the database interface for SubscriptionDocuments
type Subscriptions interface {
	Create(context.Context, *api.SubscriptionDocument) (*api.SubscriptionDocument, error)
	Get(context.Context, string) (*api.SubscriptionDocument, error)
	Update(context.Context, *api.SubscriptionDocument) (*api.SubscriptionDocument, error)
	ChangeFeed() cosmosdb.SubscriptionDocumentIterator
	Dequeue(context.Context) (*api.SubscriptionDocument, error)
	Lease(context.Context, string) (*api.SubscriptionDocument, error)
	EndLease(context.Context, string, bool, bool) (*api.SubscriptionDocument, error)
}

// NewSubscriptions returns a new Subscriptions
func NewSubscriptions(ctx context.Context, isLocalDevelopmentMode bool, dbc cosmosdb.DatabaseClient) (Subscriptions, error) {
	dbid, err := Name(isLocalDevelopmentMode)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	triggers := []*cosmosdb.Trigger{
		{
			ID:               "renewLease",
			TriggerOperation: cosmosdb.TriggerOperationAll,
			TriggerType:      cosmosdb.TriggerTypePre,
			Body: `function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	body["leaseExpires"] = Math.floor(date.getTime() / 1000) + 60;
	request.setBody(body);
}`,
		},
		{
			ID:               "retryLater",
			TriggerOperation: cosmosdb.TriggerOperationAll,
			TriggerType:      cosmosdb.TriggerTypePre,
			Body: `function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	body["leaseExpires"] = Math.floor(date.getTime() / 1000) + 600;
	request.setBody(body);
}`,
		},
	}

	triggerc := cosmosdb.NewTriggerClient(collc, collSubscriptions)
	for _, trigger := range triggers {
		_, err := triggerc.Create(ctx, trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	documentClient := cosmosdb.NewSubscriptionDocumentClient(collc, collSubscriptions)
	return NewSubscriptionsWithProvidedClient(documentClient, uuid.Must(uuid.NewV4()).String()), nil
}

func NewSubscriptionsWithProvidedClient(client cosmosdb.SubscriptionDocumentClient, uuid string) Subscriptions {
	return &subscriptions{
		c:    client,
		uuid: uuid,
	}
}

func (c *subscriptions) Create(ctx context.Context, doc *api.SubscriptionDocument) (*api.SubscriptionDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *subscriptions) Get(ctx context.Context, id string) (*api.SubscriptionDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *subscriptions) patch(ctx context.Context, id string, f func(*api.SubscriptionDocument) error, options *cosmosdb.Options) (*api.SubscriptionDocument, error) {
	var doc *api.SubscriptionDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.update(ctx, doc, options)
		return
	})

	return doc, err
}

func (c *subscriptions) patchWithLease(ctx context.Context, key string, f func(*api.SubscriptionDocument) error, options *cosmosdb.Options) (*api.SubscriptionDocument, error) {
	return c.patch(ctx, key, func(doc *api.SubscriptionDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, options)
}

func (c *subscriptions) Update(ctx context.Context, doc *api.SubscriptionDocument) (*api.SubscriptionDocument, error) {
	return c.update(ctx, doc, nil)
}

func (c *subscriptions) update(ctx context.Context, doc *api.SubscriptionDocument, options *cosmosdb.Options) (*api.SubscriptionDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, options)
}

func (c *subscriptions) ChangeFeed() cosmosdb.SubscriptionDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *subscriptions) Dequeue(ctx context.Context) (*api.SubscriptionDocument, error) {
	i := c.c.Query("", &cosmosdb.Query{Query: SubscriptionsDequeueQuery}, nil)

	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return nil, err
		}
		if docs == nil {
			return nil, nil
		}

		for _, doc := range docs.SubscriptionDocuments {
			doc.LeaseOwner = c.uuid
			doc.Dequeues++
			doc, err = c.update(ctx, doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}
}

func (c *subscriptions) Lease(ctx context.Context, id string) (*api.SubscriptionDocument, error) {
	return c.patchWithLease(ctx, id, func(doc *api.SubscriptionDocument) error {
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *subscriptions) EndLease(ctx context.Context, id string, done, retryLater bool) (*api.SubscriptionDocument, error) {
	var options *cosmosdb.Options
	if retryLater {
		options = &cosmosdb.Options{PreTriggers: []string{"retryLater"}}
	}

	return c.patchWithLease(ctx, id, func(doc *api.SubscriptionDocument) error {
		if done {
			doc.Deleting = false
		}

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		if done || retryLater {
			doc.Dequeues = 0
		}

		return nil
	}, options)
}
