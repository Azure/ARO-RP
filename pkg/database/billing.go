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
	c cosmosdb.BillingDocumentClient
}

// Billing is the database interface for BillingDocuments
type Billing interface {
	Create(context.Context, *api.BillingDocument) (*api.BillingDocument, error)
	Get(context.Context, string) (*api.BillingDocument, error)
	MarkForDeletion(context.Context, string) (*api.BillingDocument, error)
	// UpdateLastBillingTimestamp uses the caller's observed ETag and returns
	// precondition failures without retrying the timestamp against a newer version.
	UpdateLastBillingTimestamp(ctx context.Context, id, expectedETag string, timestamp int) (*api.BillingDocument, error)
	List(string) cosmosdb.BillingDocumentIterator
	ListAll(context.Context) (*api.BillingDocuments, error)
	Delete(context.Context, *api.BillingDocument) error
}

// NewBilling returns a new Billing
func NewBilling(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (Billing, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewBillingDocumentClient(collc, collBilling)
	return NewBillingWithProvidedClient(documentClient), nil
}

func NewBillingWithProvidedClient(client cosmosdb.BillingDocumentClient) Billing {
	return &billing{
		c: client,
	}
}

// Creating Billing Document
func (c *billing) Create(ctx context.Context, doc *api.BillingDocument) (*api.BillingDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, &cosmosdb.Options{PreTriggers: []string{"setCreationBillingTimeStamp"}})
}

func (c *billing) Get(ctx context.Context, id string) (*api.BillingDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *billing) patch(ctx context.Context, id string, f func(*api.BillingDocument) error, options *cosmosdb.Options) (*api.BillingDocument, error) {
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

		doc, err = c.c.Replace(ctx, doc.ID, doc, options)
		return
	})

	return doc, err
}

// patchIfMatch applies a transformation only to the caller's observed version.
// Conflicts return to the caller because a fresh read cannot validate its prior decision.
func (c *billing) patchIfMatch(ctx context.Context, id, expectedETag string, f func(*api.BillingDocument) error) (*api.BillingDocument, error) {
	if expectedETag == "" {
		return nil, cosmosdb.ErrETagRequired
	}
	if expectedETag == "*" {
		return nil, fmt.Errorf("billing document update requires a specific ETag")
	}

	doc, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("billing document %q is missing", id)
	}
	if doc.ETag != expectedETag {
		return nil, &cosmosdb.Error{
			StatusCode: http.StatusPreconditionFailed,
			Message:    fmt.Sprintf("billing document %q changed since it was read", id),
		}
	}

	err = f(doc)
	if err != nil {
		return nil, err
	}

	// The matching read does not protect against a writer racing this replacement.
	return c.c.Replace(ctx, id, doc, &cosmosdb.Options{NoETag: false})
}

// MarkForDeletion update the deletion timestamp field in the document
func (c *billing) MarkForDeletion(ctx context.Context, id string) (*api.BillingDocument, error) {
	// The trigger sets deletionTime only if unset, so retries can safely preserve
	// concurrent changes by rereading and replacing with the fresh document's ETag.
	return c.patch(ctx, id, func(billingdoc *api.BillingDocument) error {
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"setDeletionBillingTimeStamp"}})
}

// List produces and iterator for paging through all billing documents.
func (c *billing) List(continuation string) cosmosdb.BillingDocumentIterator {
	return c.c.List(&cosmosdb.Options{Continuation: continuation})
}

// ListAll list all the billing documents
func (c *billing) ListAll(ctx context.Context) (*api.BillingDocuments, error) {
	return c.c.ListAll(ctx, nil)
}

// Delete a billing document
func (c *billing) Delete(ctx context.Context, doc *api.BillingDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	// NoETag is intentional: terminal cleanup must delete unconditionally, not 412 on a concurrent write
	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

func (c *billing) UpdateLastBillingTimestamp(ctx context.Context, id, expectedETag string, timestamp int) (*api.BillingDocument, error) {
	// A fresh read cannot validate a watermark calculated from an older version;
	// the caller must reconcile conflicts before deciding whether to advance it.
	return c.patchIfMatch(ctx, id, expectedETag, func(doc *api.BillingDocument) error {
		if doc.Billing == nil {
			return fmt.Errorf("billing document %q is missing billing data", id)
		}

		doc.Billing.LastBillingTime = timestamp
		return nil
	})
}
