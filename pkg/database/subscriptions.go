package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

type subscriptions struct {
	c    cosmosdb.SubscriptionDocumentClient
	uuid uuid.UUID
}

// Subscriptions is the database interface for SubscriptionDocuments
type Subscriptions interface {
	Create(*api.SubscriptionDocument) (*api.SubscriptionDocument, error)
	Get(api.Key) (*api.SubscriptionDocument, error)
	Patch(api.Key, func(*api.SubscriptionDocument) error) (*api.SubscriptionDocument, error)
	Update(*api.SubscriptionDocument) (*api.SubscriptionDocument, error)
	Delete(*api.SubscriptionDocument) error
	Dequeue() (*api.SubscriptionDocument, error)
	Lease(api.Key) (*api.SubscriptionDocument, error)
	EndLease(api.Key, bool, bool) (*api.SubscriptionDocument, error)
}

// NewSubscriptions returns a new Subscriptions
func NewSubscriptions(ctx context.Context, uuid uuid.UUID, dbc cosmosdb.DatabaseClient, dbid, collid string) (Subscriptions, error) {
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

	triggerc := cosmosdb.NewTriggerClient(collc, collid)
	for _, trigger := range triggers {
		_, err := triggerc.Create(trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	return &subscriptions{
		c:    cosmosdb.NewSubscriptionDocumentClient(collc, collid),
		uuid: uuid,
	}, nil
}

func (c *subscriptions) Create(doc *api.SubscriptionDocument) (*api.SubscriptionDocument, error) {
	if string(doc.Key) != strings.ToLower(string(doc.Key)) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	doc, err := c.c.Create(string(doc.Key), doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *subscriptions) Get(key api.Key) (*api.SubscriptionDocument, error) {
	if string(key) != strings.ToLower(string(key)) {
		return nil, fmt.Errorf("key %q is not lower case", key)
	}

	docs, err := c.c.QueryAll(string(key), &cosmosdb.Query{
		Query: "SELECT * FROM Subscriptions doc WHERE doc.key = @key",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@key",
				Value: string(key),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	switch {
	case len(docs.SubscriptionDocuments) > 1:
		return nil, fmt.Errorf("read %d documents, expected <= 1", len(docs.SubscriptionDocuments))
	case len(docs.SubscriptionDocuments) == 1:
		return docs.SubscriptionDocuments[0], nil
	default:
		return nil, &cosmosdb.Error{StatusCode: http.StatusNotFound}
	}
}

func (c *subscriptions) Patch(key api.Key, f func(*api.SubscriptionDocument) error) (*api.SubscriptionDocument, error) {
	return c.patch(key, f, nil)
}

func (c *subscriptions) patch(key api.Key, f func(*api.SubscriptionDocument) error, options *cosmosdb.Options) (*api.SubscriptionDocument, error) {
	var doc *api.SubscriptionDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(key)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.update(doc, options)
		return
	})

	return doc, err
}

func (c *subscriptions) Update(doc *api.SubscriptionDocument) (*api.SubscriptionDocument, error) {
	return c.update(doc, nil)
}

func (c *subscriptions) update(doc *api.SubscriptionDocument, options *cosmosdb.Options) (*api.SubscriptionDocument, error) {
	if string(doc.Key) != strings.ToLower(string(doc.Key)) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Replace(string(doc.Key), doc, options)
}

func (c *subscriptions) Delete(doc *api.SubscriptionDocument) error {
	if string(doc.Key) != strings.ToLower(string(doc.Key)) {
		return fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Delete(string(doc.Key), doc, &cosmosdb.Options{NoETag: true})
}

func (c *subscriptions) Dequeue() (*api.SubscriptionDocument, error) {
	i := c.c.Query("", &cosmosdb.Query{
		Query: `SELECT * FROM Subscriptions doc WHERE (doc.deleting ?? false) AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
	})

	for {
		docs, err := i.Next()
		if err != nil {
			return nil, err
		}
		if docs == nil {
			return nil, nil
		}

		for _, doc := range docs.SubscriptionDocuments {
			doc.LeaseOwner = &c.uuid
			doc.Dequeues++
			doc, err = c.update(doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}
}

func (c *subscriptions) Lease(key api.Key) (*api.SubscriptionDocument, error) {
	return c.patch(key, func(doc *api.SubscriptionDocument) error {
		if doc.LeaseOwner == nil || !uuid.Equal(*doc.LeaseOwner, c.uuid) {
			return fmt.Errorf("lost lease")
		}
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *subscriptions) EndLease(key api.Key, done, retryLater bool) (*api.SubscriptionDocument, error) {
	var options *cosmosdb.Options
	if retryLater {
		options = &cosmosdb.Options{PreTriggers: []string{"retryLater"}}
	}

	return c.patch(key, func(doc *api.SubscriptionDocument) error {
		if doc.LeaseOwner == nil || !uuid.Equal(*doc.LeaseOwner, c.uuid) {
			return fmt.Errorf("lost lease")
		}

		if done {
			doc.Deleting = false
		}

		doc.LeaseOwner = nil
		doc.LeaseExpires = 0

		if done || retryLater {
			doc.Dequeues = 0
		}

		return nil
	}, options)
}
