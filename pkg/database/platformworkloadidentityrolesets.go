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

type platformWorkloadIdentityRoleSets struct {
	c    cosmosdb.PlatformWorkloadIdentityRoleSetDocumentClient
	uuid uuid.Generator
}

type PlatformWorkloadIdentityRoleSets interface {
	ChangeFeed() cosmosdb.PlatformWorkloadIdentityRoleSetDocumentIterator
	Create(context.Context, *api.PlatformWorkloadIdentityRoleSetDocument) (*api.PlatformWorkloadIdentityRoleSetDocument, error)
	Delete(context.Context, *api.PlatformWorkloadIdentityRoleSetDocument) error
	Get(context.Context, string) (*api.PlatformWorkloadIdentityRoleSetDocument, error)
	Update(context.Context, *api.PlatformWorkloadIdentityRoleSetDocument) (*api.PlatformWorkloadIdentityRoleSetDocument, error)
	Patch(context.Context, string, func(*api.PlatformWorkloadIdentityRoleSetDocument) error) (*api.PlatformWorkloadIdentityRoleSetDocument, error)
	ListAll(context.Context) (*api.PlatformWorkloadIdentityRoleSetDocuments, error)
	NewUUID() string
}

func NewPlatformWorkloadIdentityRoleSets(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (PlatformWorkloadIdentityRoleSets, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewPlatformWorkloadIdentityRoleSetDocumentClient(collc, collPlatformWorkloadIdentityRoleSet)
	return NewPlatformWorkloadIdentityRoleSetsWithProvidedClient(documentClient, uuid.DefaultGenerator), nil
}

func NewPlatformWorkloadIdentityRoleSetsWithProvidedClient(client cosmosdb.PlatformWorkloadIdentityRoleSetDocumentClient, uuid uuid.Generator) PlatformWorkloadIdentityRoleSets {
	return &platformWorkloadIdentityRoleSets{
		c:    client,
		uuid: uuid,
	}
}

func (c *platformWorkloadIdentityRoleSets) ChangeFeed() cosmosdb.PlatformWorkloadIdentityRoleSetDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *platformWorkloadIdentityRoleSets) Create(ctx context.Context, doc *api.PlatformWorkloadIdentityRoleSetDocument) (*api.PlatformWorkloadIdentityRoleSetDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, nil)
}

func (c *platformWorkloadIdentityRoleSets) Delete(ctx context.Context, doc *api.PlatformWorkloadIdentityRoleSetDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

func (c *platformWorkloadIdentityRoleSets) Get(ctx context.Context, id string) (*api.PlatformWorkloadIdentityRoleSetDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *platformWorkloadIdentityRoleSets) Patch(ctx context.Context, id string, f func(*api.PlatformWorkloadIdentityRoleSetDocument) error) (*api.PlatformWorkloadIdentityRoleSetDocument, error) {
	var doc *api.PlatformWorkloadIdentityRoleSetDocument

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

func (c *platformWorkloadIdentityRoleSets) Update(ctx context.Context, doc *api.PlatformWorkloadIdentityRoleSetDocument) (*api.PlatformWorkloadIdentityRoleSetDocument, error) {
	return c.update(ctx, doc)
}

func (c *platformWorkloadIdentityRoleSets) update(ctx context.Context, doc *api.PlatformWorkloadIdentityRoleSetDocument) (*api.PlatformWorkloadIdentityRoleSetDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, nil)
}

func (c *platformWorkloadIdentityRoleSets) ListAll(ctx context.Context) (*api.PlatformWorkloadIdentityRoleSetDocuments, error) {
	return c.c.ListAll(ctx, nil)
}

func (c *platformWorkloadIdentityRoleSets) NewUUID() string {
	return c.uuid.Generate()
}
