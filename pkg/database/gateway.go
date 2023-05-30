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

type gateway struct {
	c             cosmosdb.GatewayDocumentClient
	uuidGenerator uuid.Generator
}

type Gateway interface {
	ChangeFeed() cosmosdb.GatewayDocumentIterator
	Create(context.Context, *api.GatewayDocument) (*api.GatewayDocument, error)
	Delete(context.Context, *api.GatewayDocument) error
	Get(context.Context, string) (*api.GatewayDocument, error)
	Patch(context.Context, string, func(*api.GatewayDocument) error) (*api.GatewayDocument, error)
	NewUUID() string
}

func NewGateway(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (Gateway, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewGatewayDocumentClient(collc, collGateway)
	return NewGatewayWithProvidedClient(documentClient, uuid.DefaultGenerator), nil
}

func NewGatewayWithProvidedClient(client cosmosdb.GatewayDocumentClient, uuidGenerator uuid.Generator) Gateway {
	return &gateway{
		c:             client,
		uuidGenerator: uuidGenerator,
	}
}

func (c *gateway) NewUUID() string {
	return c.uuidGenerator.Generate()
}

func (c *gateway) ChangeFeed() cosmosdb.GatewayDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *gateway) Create(ctx context.Context, doc *api.GatewayDocument) (*api.GatewayDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, nil)
}

func (c *gateway) Delete(ctx context.Context, doc *api.GatewayDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

func (c *gateway) Get(ctx context.Context, id string) (*api.GatewayDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *gateway) Patch(ctx context.Context, id string, f func(*api.GatewayDocument) error) (*api.GatewayDocument, error) {
	var doc *api.GatewayDocument

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

func (c *gateway) update(ctx context.Context, doc *api.GatewayDocument) (*api.GatewayDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, nil)
}
