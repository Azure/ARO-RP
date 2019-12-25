package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type asyncOperations struct {
	c    cosmosdb.AsyncOperationDocumentClient
	uuid string
}

// AsyncOperations is the database interface for AsyncOperationDocuments
type AsyncOperations interface {
	Create(*api.AsyncOperationDocument) (*api.AsyncOperationDocument, error)
	Get(string) (*api.AsyncOperationDocument, error)
	Patch(string, func(*api.AsyncOperationDocument) error) (*api.AsyncOperationDocument, error)
}

// NewAsyncOperations returns a new AsyncOperations
func NewAsyncOperations(uuid string, dbc cosmosdb.DatabaseClient, dbid, collid string) (AsyncOperations, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	return &asyncOperations{
		c:    cosmosdb.NewAsyncOperationDocumentClient(collc, collid),
		uuid: uuid,
	}, nil
}

func (c *asyncOperations) Create(doc *api.AsyncOperationDocument) (*api.AsyncOperationDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(doc.ID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *asyncOperations) Get(id string) (*api.AsyncOperationDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(id, id)
}

func (c *asyncOperations) Patch(id string, f func(*api.AsyncOperationDocument) error) (*api.AsyncOperationDocument, error) {
	var doc *api.AsyncOperationDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.c.Replace(doc.ID, doc, nil)
		return
	})

	return doc, err
}
