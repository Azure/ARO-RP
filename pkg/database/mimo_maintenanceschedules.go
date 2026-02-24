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
	MaintenanceScheduleQueryValid = `SELECT * FROM MaintenanceSchedules doc WHERE doc.maintenanceSchedule.state IN ("Enabled", "Processing")`
)

type MaintenanceScheduleDocumentMutator func(*api.MaintenanceScheduleDocument) error

type maintenanceSchedules struct {
	c             cosmosdb.MaintenanceScheduleDocumentClient
	collc         cosmosdb.CollectionClient
	uuid          string
	uuidGenerator uuid.Generator
}

type MaintenanceSchedules interface {
	GetValid(context.Context, string) (cosmosdb.MaintenanceScheduleDocumentIterator, error)
	Create(context.Context, *api.MaintenanceScheduleDocument) (*api.MaintenanceScheduleDocument, error)
	Update(context.Context, *api.MaintenanceScheduleDocument) (*api.MaintenanceScheduleDocument, error)
	Patch(context.Context, string, MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error)
	List(string) cosmosdb.MaintenanceScheduleDocumentIterator
	Get(context.Context, string) (*api.MaintenanceScheduleDocument, error)
	Delete(context.Context, string) error

	NewUUID() string
}

func NewMaintenanceSchedules(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (MaintenanceSchedules, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewMaintenanceScheduleDocumentClient(collc, collMaintenanceSchedules)
	return NewMaintenanceSchedulesWithProvidedClient(documentClient, collc, uuid.DefaultGenerator.Generate(), uuid.DefaultGenerator), nil
}

func NewMaintenanceSchedulesWithProvidedClient(client cosmosdb.MaintenanceScheduleDocumentClient, collectionClient cosmosdb.CollectionClient, uuid string, uuidGenerator uuid.Generator) MaintenanceSchedules {
	return &maintenanceSchedules{
		c:             client,
		uuid:          uuid,
		collc:         collectionClient,
		uuidGenerator: uuidGenerator,
	}
}

func (c *maintenanceSchedules) NewUUID() string {
	return c.uuidGenerator.Generate()
}

func (c *maintenanceSchedules) GetValid(ctx context.Context, continuation string) (cosmosdb.MaintenanceScheduleDocumentIterator, error) {
	return c.c.Query("", &cosmosdb.Query{
		Query: MaintenanceScheduleQueryValid,
	}, &cosmosdb.Options{Continuation: continuation}), nil
}

func (c *maintenanceSchedules) Create(ctx context.Context, doc *api.MaintenanceScheduleDocument) (*api.MaintenanceScheduleDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *maintenanceSchedules) Get(ctx context.Context, id string) (*api.MaintenanceScheduleDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *maintenanceSchedules) List(continuation string) cosmosdb.MaintenanceScheduleDocumentIterator {
	return c.c.List(&cosmosdb.Options{Continuation: continuation})
}

func (c *maintenanceSchedules) Patch(ctx context.Context, id string, f MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error) {
	return c.patch(ctx, id, f, nil)
}

func (c *maintenanceSchedules) Update(ctx context.Context, doc *api.MaintenanceScheduleDocument) (*api.MaintenanceScheduleDocument, error) {
	return c.update(ctx, doc, nil)
}

func (c *maintenanceSchedules) patch(ctx context.Context, id string, f MaintenanceScheduleDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceScheduleDocument, error) {
	var doc *api.MaintenanceScheduleDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
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

func (c *maintenanceSchedules) update(ctx context.Context, doc *api.MaintenanceScheduleDocument, options *cosmosdb.Options) (*api.MaintenanceScheduleDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, options)
}

func (c *maintenanceSchedules) ChangeFeed() cosmosdb.MaintenanceScheduleDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *maintenanceSchedules) Delete(ctx context.Context, id string) error {
	return c.c.Delete(ctx, id, &api.MaintenanceScheduleDocument{ID: id}, nil)
}
