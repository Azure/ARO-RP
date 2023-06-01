package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type openShiftVersions struct {
	c    cosmosdb.OpenShiftVersionDocumentClient
	uuid uuid.Generator
}

type OpenShiftVersions interface {
	ChangeFeed() cosmosdb.OpenShiftVersionDocumentIterator
	Create(context.Context, *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error)
	Delete(context.Context, *api.OpenShiftVersionDocument) error
	Get(context.Context, string) (*api.OpenShiftVersionDocument, error)
	Update(context.Context, *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error)
	Patch(context.Context, string, func(*api.OpenShiftVersionDocument) error) (*api.OpenShiftVersionDocument, error)
	ListAll(context.Context) (*api.OpenShiftVersionDocuments, error)
	NewUUID() string
}

func NewOpenShiftVersions(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (OpenShiftVersions, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewOpenShiftVersionDocumentClient(collc, collOpenShiftVersion)
	return NewOpenShiftVersionsWithProvidedClient(documentClient, uuid.DefaultGenerator), nil
}

func NewOpenShiftVersionsWithProvidedClient(client cosmosdb.OpenShiftVersionDocumentClient, uuid uuid.Generator) OpenShiftVersions {
	return &openShiftVersions{
		c:    client,
		uuid: uuid,
	}
}

func (c *openShiftVersions) ChangeFeed() cosmosdb.OpenShiftVersionDocumentIterator {
	return c.c.ChangeFeed(nil)
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

func (c *openShiftVersions) Update(ctx context.Context, doc *api.OpenShiftVersionDocument) (*api.OpenShiftVersionDocument, error) {
	return c.update(ctx, doc)
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

func (c *openShiftVersions) NewUUID() string {
	return c.uuid.Generate()
}
