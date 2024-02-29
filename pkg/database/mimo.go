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
	MaintenanceManifestQueryForCluster  = `SELECT * FROM MaintenanceManifests doc WHERE doc.maintenanceManifest.state IN ("Pending", "InProgress") AND doc.clusterID = @clusterID ORDER BY doc.maintenanceManifest.runAfter`
	MaintenanceManifestQueueLengthQuery = `SELECT VALUE COUNT(1) FROM MaintenanceManifests doc WHERE doc.maintenanceManifest.state IN ("Pending") AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`
	MaintenanceManifestDequeueQuery     = `SELECT * FROM MaintenanceManifests doc WHERE doc.maintenanceManifest.state IN ("Pending") AND doc.clusterID = @clusterID AND (doc.maintenanceManifest.runAfter ?? 0) < GetCurrentTimestamp() / 1000 AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000 SORT BY doc.maintenanceManifest.runAfter asc, doc.maintenanceManifest.priority asc`
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
	GetByClusterID(context.Context, string, string) (cosmosdb.MaintenanceManifestDocumentIterator, error)
	Patch(context.Context, string, string, MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error)
	PatchWithLease(context.Context, string, string, MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error)
	Lease(context.Context, string, string) (*api.MaintenanceManifestDocument, error)
	EndLease(context.Context, string, string, api.MaintenanceManifestState, *string) (*api.MaintenanceManifestDocument, error)
	Dequeue(ctx context.Context, clusterID string) (*api.MaintenanceManifestDocument, error)

	NewUUID() string
}

func NewMaintenanceManifests(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (MaintenanceManifests, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	triggers := []*cosmosdb.Trigger{
		{
			ID:               "renewLease",
			TriggerOperation: cosmosdb.TriggerOperationAll,
			TriggerType:      cosmosdb.TriggerTypePre,
			Body: `function trigger() {
	var request = getContext().getRequest();
	var body = request.getBody();
	var date = new Date();
	body["leaseExpires"] = Math.floor(date.getTime() / 1000) + 60;
	request.setBody(body);
}`,
		},
	}

	triggerc := cosmosdb.NewTriggerClient(collc, collMaintenanceManifests)
	for _, trigger := range triggers {
		_, err := triggerc.Create(ctx, trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

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

	doc, err := c.c.Create(ctx, doc.ClusterID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *maintenanceManifests) Get(ctx context.Context, clusterID string, id string) (*api.MaintenanceManifestDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, clusterID, id, nil)
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

func (c *maintenanceManifests) Patch(ctx context.Context, clusterID string, id string, f MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error) {
	var doc *api.MaintenanceManifestDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, clusterID, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.c.Replace(ctx, doc.ClusterID, doc, nil)
		return
	})

	return doc, err
}

func (c *maintenanceManifests) patch(ctx context.Context, clusterID string, id string, f MaintenanceManifestDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceManifestDocument, error) {
	var doc *api.MaintenanceManifestDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, clusterID, id)
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

func (c *maintenanceManifests) PatchWithLease(ctx context.Context, clusterID string, id string, f MaintenanceManifestDocumentMutator) (*api.MaintenanceManifestDocument, error) {
	return c.patchWithLease(ctx, clusterID, id, f, nil)
}

func (c *maintenanceManifests) patchWithLease(ctx context.Context, clusterID string, id string, f MaintenanceManifestDocumentMutator, options *cosmosdb.Options) (*api.MaintenanceManifestDocument, error) {
	return c.patch(ctx, clusterID, id, func(doc *api.MaintenanceManifestDocument) error {
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

	return c.c.Replace(ctx, doc.ClusterID, doc, options)
}

func (c *maintenanceManifests) ChangeFeed() cosmosdb.MaintenanceManifestDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *maintenanceManifests) GetByClusterID(ctx context.Context, clusterID string, continuation string) (cosmosdb.MaintenanceManifestDocumentIterator, error) {
	if clusterID != strings.ToLower(clusterID) {
		return nil, fmt.Errorf("clusterID %q is not lower case", clusterID)
	}

	return c.c.Query("", &cosmosdb.Query{
		Query: MaintenanceManifestQueryForCluster,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@clusterID",
				Value: clusterID,
			},
		},
	}, &cosmosdb.Options{Continuation: continuation}), nil
}

func (c *maintenanceManifests) Lease(ctx context.Context, clusterID string, id string) (*api.MaintenanceManifestDocument, error) {
	return c.patchWithLease(ctx, clusterID, id, func(doc *api.MaintenanceManifestDocument) error {
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *maintenanceManifests) EndLease(ctx context.Context, clusterID string, id string, provisioningState api.MaintenanceManifestState, statusString *string) (*api.MaintenanceManifestDocument, error) {
	return c.patchWithLease(ctx, clusterID, id, func(doc *api.MaintenanceManifestDocument) error {
		doc.MaintenanceManifest.State = provisioningState
		if statusString != nil {
			doc.MaintenanceManifest.StatusText = *statusString
		}

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		if provisioningState != api.MaintenanceManifestStateFailed {
			doc.Dequeues = 0
		}
		return nil
	}, nil)
}

func (c *maintenanceManifests) Dequeue(ctx context.Context, clusterID string) (*api.MaintenanceManifestDocument, error) {
	if clusterID != strings.ToLower(clusterID) {
		return nil, fmt.Errorf("clusterID %q is not lower case", clusterID)
	}
	i := c.c.Query(clusterID, &cosmosdb.Query{
		Query: MaintenanceManifestDequeueQuery,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@clusterID",
				Value: clusterID,
			},
		},
	}, nil)

	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return nil, err
		}
		if docs == nil {
			return nil, nil
		}

		for _, doc := range docs.MaintenanceManifestDocuments {
			doc.LeaseOwner = c.uuid
			doc.Dequeues++
			doc, err = c.update(ctx, doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}
}
