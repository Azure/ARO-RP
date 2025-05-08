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

type asyncOperations struct {
	c             cosmosdb.AsyncOperationDocumentClient
	uuidGenerator uuid.Generator
}

// AsyncOperations is the database interface for AsyncOperationDocuments
type AsyncOperations interface {
	Create(context.Context, *api.AsyncOperationDocument) (*api.AsyncOperationDocument, error)
	Get(context.Context, string) (*api.AsyncOperationDocument, error)
	Patch(context.Context, string, func(*api.AsyncOperationDocument) error) (*api.AsyncOperationDocument, error)
	NewUUID() string
}

// NewAsyncOperations returns a new AsyncOperations
func NewAsyncOperations(ctx context.Context, isLocalDevelopmentMode bool, dbc cosmosdb.DatabaseClient, dbName string) (AsyncOperations, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)
	client := cosmosdb.NewAsyncOperationDocumentClient(collc, collAsyncOperations)
	return NewAsyncOperationsWithProvidedClient(client, uuid.DefaultGenerator), nil
}

func NewAsyncOperationsWithProvidedClient(client cosmosdb.AsyncOperationDocumentClient, uuidGenerator uuid.Generator) AsyncOperations {
	return &asyncOperations{
		c:             client,
		uuidGenerator: uuidGenerator,
	}
}

func (c *asyncOperations) NewUUID() string {
	return c.uuidGenerator.Generate()
}

func (c *asyncOperations) Create(ctx context.Context, doc *api.AsyncOperationDocument) (*api.AsyncOperationDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *asyncOperations) Get(ctx context.Context, id string) (*api.AsyncOperationDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *asyncOperations) Patch(ctx context.Context, id string, f func(*api.AsyncOperationDocument) error) (*api.AsyncOperationDocument, error) {
	var doc *api.AsyncOperationDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.c.Replace(ctx, doc.ID, doc, nil)
		return
	})

	return doc, err
}
