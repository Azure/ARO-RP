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

const (
	PoolWorkerGetMasterQuery  string = `SELECT * FROM PoolWorkers doc WHERE doc.id = "@workerType" AND doc.poolWorker.workerType = "@workerType" AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`
	PoolWorkerGetWorkersQuery string = `SELECT * FROM PoolWorkers doc WHERE doc.id != "@workerType" AND doc.poolWorker.workerType = "@workerType"`
)

type poolWorkers struct {
	c    cosmosdb.PoolWorkerDocumentClient
	uuid string
}

// PoolWorkers is the database interface for PoolWorkerDocuments
type PoolWorkers interface {
	Create(context.Context, api.PoolWorkerType, *api.PoolWorkerDocument) (*api.PoolWorkerDocument, error)
	PatchWithLease(context.Context, api.PoolWorkerType, string, func(*api.PoolWorkerDocument) error) (*api.PoolWorkerDocument, error)
	TryLease(context.Context, api.PoolWorkerType) (*api.PoolWorkerDocument, error)
	ListBuckets(context.Context, api.PoolWorkerType) ([]int, error)
	ListPoolWorkers(context.Context, api.PoolWorkerType) (*api.PoolWorkerDocuments, error)
	PoolWorkerHeartbeat(context.Context, api.PoolWorkerType, int) error
}

// NewPoolWorkers returns a new PoolWorkers
func NewPoolWorkers(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (PoolWorkers, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	return &poolWorkers{
		c:    cosmosdb.NewPoolWorkerDocumentClient(collc, collPoolWorkers),
		uuid: uuid.DefaultGenerator.Generate(),
	}, nil
}

func NewPoolWorkersWithProvidedClient(client cosmosdb.PoolWorkerDocumentClient, uuid string) PoolWorkers {
	return &poolWorkers{
		c:    client,
		uuid: uuid,
	}
}

func (c *poolWorkers) Create(ctx context.Context, poolWorkerType api.PoolWorkerType, doc *api.PoolWorkerDocument) (*api.PoolWorkerDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, string(poolWorkerType), doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *poolWorkers) get(ctx context.Context, poolWorkerType api.PoolWorkerType, id string) (*api.PoolWorkerDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, string(poolWorkerType), id, nil)
}

func (c *poolWorkers) patch(ctx context.Context, poolWorkerType api.PoolWorkerType, id string, f func(*api.PoolWorkerDocument) error, options *cosmosdb.Options) (*api.PoolWorkerDocument, error) {
	var doc *api.PoolWorkerDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.get(ctx, poolWorkerType, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.update(ctx, poolWorkerType, doc, options)
		return
	})

	return doc, err
}

func (c *poolWorkers) PatchWithLease(ctx context.Context, poolWorkerType api.PoolWorkerType, id string, f func(*api.PoolWorkerDocument) error) (*api.PoolWorkerDocument, error) {
	return c.patch(ctx, poolWorkerType, id, func(doc *api.PoolWorkerDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *poolWorkers) update(ctx context.Context, poolWorkerType api.PoolWorkerType, doc *api.PoolWorkerDocument, options *cosmosdb.Options) (*api.PoolWorkerDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, string(poolWorkerType), doc, options)
}

func (c *poolWorkers) TryLease(ctx context.Context, workerType api.PoolWorkerType) (*api.PoolWorkerDocument, error) {
	docs, err := c.c.QueryAll(ctx, string(workerType), &cosmosdb.Query{
		Query: PoolWorkerGetMasterQuery,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@workerType",
				Value: string(workerType),
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		return nil, nil
	}

	for _, doc := range docs.PoolWorkerDocuments {
		doc.LeaseOwner = c.uuid
		doc, err = c.update(ctx, workerType, doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
		if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
			continue
		}
		return doc, err
	}

	return nil, nil
}

func (c *poolWorkers) ListBuckets(ctx context.Context, poolWorkerType api.PoolWorkerType) (buckets []int, err error) {
	doc, err := c.get(ctx, poolWorkerType, string(poolWorkerType))
	if err != nil || doc == nil || doc.PoolWorker == nil {
		return nil, err
	}

	for i, poolworker := range doc.PoolWorker.Buckets {
		if poolworker == c.uuid {
			buckets = append(buckets, i)
		}
	}

	return buckets, nil
}

func (c *poolWorkers) ListPoolWorkers(ctx context.Context, poolWorkerType api.PoolWorkerType) (*api.PoolWorkerDocuments, error) {
	return c.c.QueryAll(ctx, string(poolWorkerType), &cosmosdb.Query{
		Query: PoolWorkerGetWorkersQuery,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@workerType",
				Value: string(poolWorkerType),
			},
		},
	}, nil)
}

func (c *poolWorkers) PoolWorkerHeartbeat(ctx context.Context, poolWorkerType api.PoolWorkerType, ttl int) error {
	doc := &api.PoolWorkerDocument{
		ID:         c.uuid,
		WorkerType: poolWorkerType,
		TTL:        ttl,
	}
	_, err := c.update(ctx, poolWorkerType, doc, &cosmosdb.Options{NoETag: true})
	if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		_, err = c.Create(ctx, poolWorkerType, doc)
	}
	return err
}
