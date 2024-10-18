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
	MaintenanceManifestDequeueQueryForCluster = `SELECT * FROM MaintenanceManifests doc WHERE doc.maintenanceManifest.state IN ("Pending") AND doc.clusterResourceID = @clusterResourceID`
	MaintenanceManifestQueryForCluster        = `SELECT * FROM MaintenanceManifests doc WHERE doc.clusterResourceID = @clusterResourceID`
	MaintenanceManifestQueueLengthQuery       = `SELECT VALUE COUNT(1) FROM MaintenanceManifests doc WHERE doc.maintenanceManifest.state IN ("Pending") AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`
)

type MaintenanceManifestDocumentMutator func(*api.MaintenanceManifestDocument) error

type maintenanceManifests struct {
	c             cosmosdb.MaintenanceManifestDocumentClient
	collc         cosmosdb.CollectionClient
	uuid          string
	uuidGenerator uuid.Generator
}

type MaintenanceManifests interface {
	Create(context.Context, *api.MaintenanceManifestDocument) (*api.MaintenanceManifestDocument, error)
	GetByClusterResourceID(ctx context.Context, clusterResourceID string, continuation string) (cosmosdb.MaintenanceManifestDocumentIterator, error)
	GetQueuedByClusterResourceID(ctx context.Context, clusterResourceID string, continuation string) (cosmosdb.MaintenanceManifestDocumentIterator, error)
	Patch(context.Context, string, string, MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error)
	PatchWithLease(context.Context, string, string, MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error)
	Lease(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceManifestDocument, error)
	EndLease(context.Context, string, string, api.MaintenanceManifestState, *string) (*api.MaintenanceManifestDocument, error)
	Get(context.Context, string, string) (*api.MaintenanceManifestDocument, error)
	Delete(context.Context, string, string) error

	NewUUID() string
}

func NewMaintenanceManifests(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (MaintenanceManifests, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	documentClient := cosmosdb.NewMaintenanceManifestDocumentClient(collc, collMaintenanceManifests)
	return NewMaintenanceManifestsWithProvidedClient(documentClient, collc, uuid.DefaultGenerator.Generate(), uuid.DefaultGenerator), nil
}

func NewMaintenanceManifestsWithProvidedClient(client cosmosdb.MaintenanceManifestDocumentClient, collectionClient cosmosdb.CollectionClient, uuid string, uuidGenerator uuid.Generator) MaintenanceManifests {
	return &maintenanceManifests{
		c:             client,
		uuid:          uuid,
		collc:         collectionClient,
		uuidGenerator: uuidGenerator,
	}
}

func (c *maintenanceManifests) NewUUID() string {
	return c.uuidGenerator.Generate()
}

func (c *maintenanceManifests) Create(ctx context.Context, doc *api.MaintenanceManifestDocument) (*api.MaintenanceManifestDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	doc, err := c.c.Create(ctx, doc.ClusterResourceID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *maintenanceManifests) Get(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceManifestDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, clusterResourceID, id, nil)
}

// QueueLength returns maintenanceManifests un-queued document count.
// If error occurs, 0 is returned with error message
func (c *maintenanceManifests) QueueLength(ctx context.Context, collid string) (int, error) {
	partitions, err := c.collc.PartitionKeyRanges(ctx, collid)
	if err != nil {
		return 0, err
	}

	var countTotal int
	for _, r := range partitions.PartitionKeyRanges {
		result := c.c.Query("", &cosmosdb.Query{
			Query: MaintenanceManifestQueueLengthQuery,
		}, &cosmosdb.Options{
			PartitionKeyRangeID: r.ID,
		})
		// because we aggregate count we don't expect pagination in this query result,
		// so we gonna call Next() only once per partition.
		var data struct {
			api.MissingFields
			Document []int `json:"Documents,omitempty"`
		}
		err := result.NextRaw(ctx, -1, &data)
		if err != nil {
			return 0, err
		}

		countTotal = countTotal + data.Document[0]
	}
	return countTotal, nil
}

func (c *maintenanceManifests) Patch(ctx context.Context, clusterResourceID string, id string, f MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error) {
	return c.patch(ctx, clusterResourceID, id, f, nil)
}

func (c *maintenanceManifests) patch(ctx context.Context, clusterResourceID string, id string, f MaintenanceManifestDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceManifestDocument, error) {
	var doc *api.MaintenanceManifestDocument

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
func (c *maintenanceManifests) PatchWithLease(ctx context.Context, clusterResourceID string, id string, f MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error) {
	return c.patchWithLease(ctx, clusterResourceID, id, f, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *maintenanceManifests) patchWithLease(ctx context.Context, clusterResourceID string, id string, f MaintenanceManifestDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceManifestDocument, error) {
	return c.patch(ctx, clusterResourceID, id, func(doc *api.MaintenanceManifestDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, options)
}

func (c *maintenanceManifests) update(ctx context.Context, doc *api.MaintenanceManifestDocument, options *cosmosdb.Options) (*api.MaintenanceManifestDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ClusterResourceID, doc, options)
}

func (c *maintenanceManifests) ChangeFeed() cosmosdb.MaintenanceManifestDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *maintenanceManifests) GetByClusterResourceID(ctx context.Context, clusterResourceID string, continuation string) (cosmosdb.MaintenanceManifestDocumentIterator, error) {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return nil, fmt.Errorf("clusterResourceID %q is not lower case", clusterResourceID)
	}

	return c.c.Query("", &cosmosdb.Query{
		Query: MaintenanceManifestQueryForCluster,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@clusterResourceID",
				Value: clusterResourceID,
			},
		},
	}, &cosmosdb.Options{Continuation: continuation}), nil
}

func (c *maintenanceManifests) GetQueuedByClusterResourceID(ctx context.Context, clusterResourceID string, continuation string) (cosmosdb.MaintenanceManifestDocumentIterator, error) {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return nil, fmt.Errorf("clusterResourceID %q is not lower case", clusterResourceID)
	}

	return c.c.Query("", &cosmosdb.Query{
		Query: MaintenanceManifestDequeueQueryForCluster,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@clusterResourceID",
				Value: clusterResourceID,
			},
		},
	}, &cosmosdb.Options{Continuation: continuation}), nil
}

func (c *maintenanceManifests) EndLease(ctx context.Context, clusterResourceID string, id string, provisioningState api.MaintenanceManifestState, statusString *string) (*api.MaintenanceManifestDocument, error) {
	return c.patchWithLease(ctx, clusterResourceID, id, func(doc *api.MaintenanceManifestDocument) error {
		doc.MaintenanceManifest.State = provisioningState
		if statusString != nil {
			doc.MaintenanceManifest.StatusText = *statusString
		}

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		return nil
	}, nil)
}

// Lease performs the initial lease/dequeue on the document.
func (c *maintenanceManifests) Lease(ctx context.Context, clusterResourceID string, id string) (*api.MaintenanceManifestDocument, error) {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return nil, fmt.Errorf("clusterID %q is not lower case", clusterResourceID)
	}

	return c.patch(ctx, clusterResourceID, id, func(doc *api.MaintenanceManifestDocument) error {
		doc.LeaseOwner = c.uuid
		doc.Dequeues++
		doc.MaintenanceManifest.State = api.MaintenanceManifestStateInProgress
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *maintenanceManifests) Delete(ctx context.Context, clusterResourceID string, id string) error {
	if clusterResourceID != strings.ToLower(clusterResourceID) {
		return fmt.Errorf("clusterID %q is not lower case", clusterResourceID)
	}

	return c.c.Delete(ctx, clusterResourceID, &api.MaintenanceManifestDocument{ID: id}, nil)
}
