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

type ocmClusterDocument struct {
	c cosmosdb.OpenShiftClusterDocumentClient
}

type OCMClusterDocument interface {
	Create(context.Context, *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Get(context.Context, string) (*api.OpenShiftClusterDocument, error)
	Patch(context.Context, string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	Delete(context.Context, *api.OpenShiftClusterDocument) error
	ChangeFeed() cosmosdb.OpenShiftClusterDocumentIterator
}

func NewOCMClusterDocument(ctx context.Context, isDevelopmentMode bool, dbc cosmosdb.DatabaseClient) (OCMClusterDocument, error) {
	dbid, err := Name(isDevelopmentMode)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	documentClient := cosmosdb.NewOpenShiftClusterDocumentClient(collc, collHiveResources)
	return NewOCMClusterDocumentWithProvidedClient(documentClient), nil
}

func NewOCMClusterDocumentWithProvidedClient(client cosmosdb.OpenShiftClusterDocumentClient) OCMClusterDocument {
	return &ocmClusterDocument{c: client}
}

// Only used internally by Patch()
func (c *ocmClusterDocument) replace(ctx context.Context, doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, nil)
}

func (c *ocmClusterDocument) Create(ctx context.Context, doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, nil)
}

func (c *ocmClusterDocument) Get(ctx context.Context, id string) (*api.OpenShiftClusterDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *ocmClusterDocument) Patch(ctx context.Context, id string, callback func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	var doc *api.OpenShiftClusterDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		doc, err := c.Get(ctx, id)
		if err != nil {
			return err
		}

		err = callback(doc)
		if err != nil {
			return err
		}

		doc, err = c.replace(ctx, doc)
		return err
	})

	return doc, err
}

func (c *ocmClusterDocument) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

func (c *ocmClusterDocument) ChangeFeed() cosmosdb.OpenShiftClusterDocumentIterator {
	return c.c.ChangeFeed(nil)
}
