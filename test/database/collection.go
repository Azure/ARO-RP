package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

// fakeCollectionClient is a fake collection client for faking out PartitionKeyRanges
type fakeCollectionClient struct {
}

func (c *fakeCollectionClient) Create(ctx context.Context, newcoll *cosmosdb.Collection) (coll *cosmosdb.Collection, err error) {
	return nil, cosmosdb.ErrNotImplemented
}

func (c *fakeCollectionClient) List() cosmosdb.CollectionIterator {
	return nil
}

func (c *fakeCollectionClient) ListAll(ctx context.Context) (*cosmosdb.Collections, error) {
	return nil, cosmosdb.ErrNotImplemented
}

func (c *fakeCollectionClient) Get(ctx context.Context, collid string) (coll *cosmosdb.Collection, err error) {
	return nil, cosmosdb.ErrNotImplemented
}

func (c *fakeCollectionClient) Delete(ctx context.Context, coll *cosmosdb.Collection) error {
	return cosmosdb.ErrNotImplemented
}

func (c *fakeCollectionClient) Replace(ctx context.Context, newcoll *cosmosdb.Collection) (coll *cosmosdb.Collection, err error) {
	return nil, cosmosdb.ErrNotImplemented
}

func (c *fakeCollectionClient) PartitionKeyRanges(ctx context.Context, collid string) (*cosmosdb.PartitionKeyRanges, error) {
	return &cosmosdb.PartitionKeyRanges{
		Count:      1,
		ResourceID: collid,
		PartitionKeyRanges: []cosmosdb.PartitionKeyRange{
			{
				ID: "singular",
			},
		},
	}, nil
}
