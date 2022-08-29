package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	BackendsLeaseTTL = 60 // seconds

	backendDocID        string = "state"
	renewLeaseTriggerID string = "renewLease"
)

type backends struct {
	c    cosmosdb.BackendDocumentClient
	uuid string
}

// Backends is the database interface for BackendDocuments
type Backends interface {
	Initialize(context.Context) error
	TryLease(context.Context) (*api.BackendDocument, error)
	PatchWithLease(context.Context, func(*api.BackendDocument) error) (*api.BackendDocument, error)
}

func renewLeaseTriggerOptions() *cosmosdb.Options {
	return &cosmosdb.Options{PreTriggers: []string{renewLeaseTriggerID}}
}

func NewBackends(ctx context.Context, isLocalDevelopmentMode bool, dbClient cosmosdb.DatabaseClient) (Backends, error) {
	dbid, err := Name(isLocalDevelopmentMode)
	if err != nil {
		return nil, err
	}

	collClient := cosmosdb.NewCollectionClient(dbClient, dbid)
	triggerClient := cosmosdb.NewTriggerClient(collClient, collBackends)

	trigger := &cosmosdb.Trigger{
		ID:               renewLeaseTriggerID,
		TriggerOperation: cosmosdb.TriggerOperationAll,
		TriggerType:      cosmosdb.TriggerTypePre,
		Body: fmt.Sprintf(`function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	body["leaseExpires"] = Math.floor(date.getTime() / 1000) + %d;
	request.setBody(body);
}`, BackendsLeaseTTL),
	}

	_, err = triggerClient.Create(ctx, trigger)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return nil, err
	}

	return &backends{
		c:    cosmosdb.NewBackendDocumentClient(collClient, collBackends),
		uuid: uuid.DefaultGenerator.Generate(),
	}, nil
}

func (c *backends) Initialize(ctx context.Context) error {
	doc := &api.BackendDocument{ID: backendDocID}
	_, err := c.c.Create(ctx, backendDocID, doc, nil)

	if cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		err = nil
	}

	return err
}

func (c *backends) TryLease(ctx context.Context) (*api.BackendDocument, error) {
	// Matching "doc.leaseOwner" in the SELECT query allows the current lease
	// owner to refetch the document before the lease has expired and thereby
	// renew the lease through the "renewLease" trigger function.
	docs, err := c.c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: fmt.Sprintf(`SELECT * FROM Backends doc WHERE doc.id = "%s" AND (doc.leaseOwner = "%s" OR (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000)`, backendDocID, c.uuid),
	}, nil)
	if err != nil {
		return nil, err
	}

	if docs != nil {
		for _, doc := range docs.BackendDocuments {
			doc.LeaseOwner = c.uuid
			doc, err = c.c.Replace(ctx, doc.ID, doc, renewLeaseTriggerOptions())
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}

	return nil, nil
}

func (c *backends) PatchWithLease(ctx context.Context, cb func(*api.BackendDocument) error) (*api.BackendDocument, error) {
	var doc *api.BackendDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.c.Get(ctx, backendDocID, backendDocID, nil)
		if err != nil {
			return
		}

		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		err = cb(doc)
		if err != nil {
			return
		}

		doc.ID = backendDocID
		doc, err = c.c.Replace(ctx, backendDocID, doc, renewLeaseTriggerOptions())
		return
	})

	return doc, err
}
