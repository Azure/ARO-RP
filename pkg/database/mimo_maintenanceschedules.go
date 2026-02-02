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

type MaintenanceScheduleDocumentMutator func(*api.MaintenanceScheduleDocument) error

type maintenanceSchedules struct {
	c             cosmosdb.MaintenanceScheduleDocumentClient
	collc         cosmosdb.CollectionClient
	uuid          string
	uuidGenerator uuid.Generator
}

type MaintenanceSchedules interface {
	Create(context.Context, *api.MaintenanceScheduleDocument) (*api.MaintenanceScheduleDocument, error)
	Patch(context.Context, string, string, MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error)
	PatchWithLease(context.Context, string, string, MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error)
	Lease(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceScheduleDocument, error)
	EndLease(context.Context, string, string, api.MaintenanceScheduleState) (*api.MaintenanceScheduleDocument, error)
	Get(context.Context, string, string) (*api.MaintenanceScheduleDocument, error)
	Delete(context.Context, string, string) error

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

func (c *maintenanceSchedules) Get(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceScheduleDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, clusterResourceID, id, nil)
}

func (c *maintenanceSchedules) Patch(ctx context.Context, clusterResourceID string, id string, f MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error) {
	return c.patch(ctx, clusterResourceID, id, f, nil)
}

func (c *maintenanceSchedules) patch(ctx context.Context, clusterResourceID string, id string, f MaintenanceScheduleDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceScheduleDocument, error) {
	var doc *api.MaintenanceScheduleDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, clusterResourceID, id)
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

// PatchWithLease performs a patch on the cluster that verifies the lease is
// being held by this client before applying.
func (c *maintenanceSchedules) PatchWithLease(ctx context.Context, clusterResourceID string, id string, f MaintenanceScheduleDocumentMutator) (*api.MaintenanceScheduleDocument, error) {
	return c.patchWithLease(ctx, clusterResourceID, id, f, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *maintenanceSchedules) patchWithLease(ctx context.Context, clusterResourceID string, id string, f MaintenanceScheduleDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceScheduleDocument, error) {
	return c.patch(ctx, clusterResourceID, id, func(doc *api.MaintenanceScheduleDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, options)
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

func (c *maintenanceSchedules) EndLease(ctx context.Context, clusterResourceID string, id string, provisioningState api.MaintenanceScheduleState) (*api.MaintenanceScheduleDocument, error) {
	return c.patchWithLease(ctx, clusterResourceID, id, func(doc *api.MaintenanceScheduleDocument) error {
		doc.MaintenanceSchedule.State = provisioningState

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		return nil
	}, nil)
}

// Lease performs the initial lease/dequeue on the document.
func (c *maintenanceSchedules) Lease(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceScheduleDocument, error) {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return nil, fmt.Errorf("clusterID %q is not lower case", clusterResourceID)
	}

	return c.patch(ctx, clusterResourceID, id, func(doc *api.MaintenanceScheduleDocument) error {
		doc.LeaseOwner = c.uuid
		doc.Dequeues++
		doc.MaintenanceSchedule.State = api.MaintenanceScheduleStateProcessing
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *maintenanceSchedules) Delete(ctx context.Context, clusterResourceID string, id string) error {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return fmt.Errorf("clusterID %q is not lower case", clusterResourceID)
	}

	return c.c.Delete(ctx, clusterResourceID, &api.MaintenanceScheduleDocument{ID: id}, nil)
}
