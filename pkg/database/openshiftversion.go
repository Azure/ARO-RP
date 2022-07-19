package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type openShiftVersions struct {
	c cosmosdb.OpenShiftVersionDocumentClient
}

type OpenShiftVersions interface {
	Create(context.Context, *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error)
	Delete(context.Context, *api.OpenShiftVersionDocument) error
	Get(context.Context, string) (*api.OpenShiftVersionDocument, error)
	Patch(context.Context, string, func(*api.OpenShiftVersionDocument) error) (*api.OpenShiftVersionDocument, error)
	ListAll(context.Context) (*api.OpenShiftVersionDocuments, error)
}

func NewOpenShiftVersions(ctx context.Context, isDevelopmentMode bool, dbc cosmosdb.DatabaseClient) (OpenShiftVersions, error) {
	dbid, err := Name(isDevelopmentMode)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	documentClient := cosmosdb.NewOpenShiftVersionDocumentClient(collc, collOpenShiftVersion)
	return NewOpenShiftVersionsWithProvidedClient(documentClient), nil
}

func NewOpenShiftVersionsWithProvidedClient(client cosmosdb.OpenShiftVersionDocumentClient) OpenShiftVersions {
	return &openShiftVersions{
		c: client,
	}
}

func (c *openShiftVersions) Create(ctx context.Context, doc *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, nil)
}

func (c *openShiftVersions) Delete(ctx context.Context, doc *api.OpenShiftVersionDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

func (c *openShiftVersions) Get(ctx context.Context, id string) (*api.OpenShiftVersionDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *openShiftVersions) Patch(ctx context.Context, id string, f func(*api.OpenShiftVersionDocument) error) (*api.OpenShiftVersionDocument, error) {
	var doc *api.OpenShiftVersionDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.update(ctx, doc)
		return
	})

	return doc, err
}

func (c *openShiftVersions) update(ctx context.Context, doc *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, nil)
}

func (c *openShiftVersions) ListAll(ctx context.Context) (*api.OpenShiftVersionDocuments, error) {
	return c.c.ListAll(ctx, nil)
}
