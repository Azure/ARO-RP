package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type billing struct {
	c    cosmosdb.BillingDocumentClient
	uuid string
}

// Billing is the database interface for BillingDocuments
type Billing interface {
	Create(context.Context, *api.BillingDocument) (*api.BillingDocument, error)
	Get(context.Context, string) (*api.BillingDocument, error)
	Patch(context.Context, string, func(*api.BillingDocument) error) (*api.BillingDocument, error)
}

// NewBilling returns a new Billing
func NewBilling(ctx context.Context, uuid string, dbc cosmosdb.DatabaseClient, dbid, collid string) (Billing, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	triggers := []*cosmosdb.Trigger{
		{
			ID:               "setCreationTimeStamp",
			TriggerOperation: cosmosdb.TriggerOperationAll,
			TriggerType:      cosmosdb.TriggerTypePre,
			Body: `function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	var now = Math.floor(date.getTime() / 1000)
	var billingBody = body["billing"]
	billingBody["creationTime"] = now;
	billingBody["lastBillingTime"] = now;
	request.setBody(body);
}`,
		},
		{
			ID:               "setDeletionTimeStamp",
			TriggerOperation: cosmosdb.TriggerOperationAll,
			TriggerType:      cosmosdb.TriggerTypePre,
			Body: `function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	var now = Math.floor(date.getTime() / 1000)
	var billingBody = body["billing"]
	billingBody["deletionTime"] = now;
	request.setBody(body);
}`,
		},
	}

	triggerc := cosmosdb.NewTriggerClient(collc, collid)
	for _, trigger := range triggers {
		_, err := triggerc.Create(ctx, trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	return &billing{
		c:    cosmosdb.NewBillingDocumentClient(collc, collid),
		uuid: uuid,
	}, nil
}

// Creating Billing Document or update existing one
func (c *billing) Create(ctx context.Context, doc *api.BillingDocument) (*api.BillingDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ID, doc, &cosmosdb.Options{PreTriggers: []string{"setCreationTimeStamp"}})

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		doc, err := c.Get(ctx, doc.ID)
		if err != nil {
			return nil, fmt.Errorf("cannot Get Billing document : %s", doc.ID)
		}

		doc, err = c.c.Replace(ctx, doc.ID, doc, &cosmosdb.Options{PreTriggers: []string{"setCreationTimeStamp"}})
	}

	return doc, err
}

func (c *billing) Get(ctx context.Context, id string) (*api.BillingDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *billing) Patch(ctx context.Context, id string, f func(*api.BillingDocument) error) (*api.BillingDocument, error) {
	var doc *api.BillingDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.c.Replace(ctx, doc.ID, doc, &cosmosdb.Options{PreTriggers: []string{"setDeletionTimeStamp"}})
		return
	})

	return doc, err
}
