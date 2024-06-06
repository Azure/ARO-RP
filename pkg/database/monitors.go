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
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type monitors struct {
	c    cosmosdb.MonitorDocumentClient
	uuid string
}

// Monitors is the database interface for MonitorDocuments
type Monitors interface {
	Create(context.Context, *api.MonitorDocument) (*api.MonitorDocument, error)
	PatchWithLease(context.Context, string, func(*api.MonitorDocument) error) (*api.MonitorDocument, error)
	TryLease(context.Context) (*api.MonitorDocument, error)
	ListBuckets(context.Context) ([]int, error)
	ListMonitors(context.Context) (*api.MonitorDocuments, error)
	MonitorHeartbeat(context.Context) error
}

// NewMonitors returns a new Monitors
func NewMonitors(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (Monitors, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	return &monitors{
		c:    cosmosdb.NewMonitorDocumentClient(collc, collMonitors),
		uuid: uuid.DefaultGenerator.Generate(),
	}, nil
}

func (c *monitors) Create(ctx context.Context, doc *api.MonitorDocument) (*api.MonitorDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *monitors) get(ctx context.Context, id string) (*api.MonitorDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *monitors) patch(ctx context.Context, id string, f func(*api.MonitorDocument) error, options *cosmosdb.Options) (*api.MonitorDocument, error) {
	var doc *api.MonitorDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.get(ctx, id)
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

func (c *monitors) PatchWithLease(ctx context.Context, id string, f func(*api.MonitorDocument) error) (*api.MonitorDocument, error) {
	return c.patch(ctx, id, func(doc *api.MonitorDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *monitors) update(ctx context.Context, doc *api.MonitorDocument, options *cosmosdb.Options) (*api.MonitorDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, options)
}

func (c *monitors) TryLease(ctx context.Context) (*api.MonitorDocument, error) {
	docs, err := c.c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: `SELECT * FROM Monitors doc WHERE doc.id = "master" AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
	}, nil)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		return nil, nil
	}

	for _, doc := range docs.MonitorDocuments {
		doc.LeaseOwner = c.uuid
		doc, err = c.update(ctx, doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
		if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
			continue
		}
		return doc, err
	}

	return nil, nil
}

func (c *monitors) ListBuckets(ctx context.Context) (buckets []int, err error) {
	doc, err := c.get(ctx, "master")
	if err != nil || doc == nil {
		return nil, err
	}

	for i, monitor := range doc.Monitor.Buckets {
		if monitor == c.uuid {
			buckets = append(buckets, i)
		}
	}

	return buckets, nil
}

func (c *monitors) ListMonitors(ctx context.Context) (*api.MonitorDocuments, error) {
	return c.c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: `SELECT * FROM Monitors doc WHERE doc.id != "master"`,
	}, nil)
}

func (c *monitors) MonitorHeartbeat(ctx context.Context) error {
	doc := &api.MonitorDocument{
		ID:  c.uuid,
		TTL: 60,
	}
	_, err := c.update(ctx, doc, &cosmosdb.Options{NoETag: true})
	if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		_, err = c.Create(ctx, doc)
	}
	return err
}
