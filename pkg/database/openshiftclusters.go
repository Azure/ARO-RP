package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type openShiftClusters struct {
	c     cosmosdb.OpenShiftClusterDocumentClient
	collc cosmosdb.CollectionClient
	uuid  string
}

// OpenShiftClusters is the database interface for OpenShiftClusterDocuments
type OpenShiftClusters interface {
	Create(context.Context, *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Get(context.Context, string) (*api.OpenShiftClusterDocument, error)
	QueueLength(context.Context, string) (int, error)
	Patch(context.Context, string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	PatchWithLease(context.Context, string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	Update(context.Context, *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Delete(context.Context, *api.OpenShiftClusterDocument) error
	ChangeFeed() cosmosdb.OpenShiftClusterDocumentIterator
	List(string) cosmosdb.OpenShiftClusterDocumentIterator
	ListByPrefix(string, string, string) (cosmosdb.OpenShiftClusterDocumentIterator, error)
	Dequeue(context.Context) (*api.OpenShiftClusterDocument, error)
	Lease(context.Context, string) (*api.OpenShiftClusterDocument, error)
	EndLease(context.Context, string, api.ProvisioningState, api.ProvisioningState, *string) (*api.OpenShiftClusterDocument, error)
	GetByClientID(ctx context.Context, partitionKey, clientID string) (*api.OpenShiftClusterDocuments, error)
	GetByClusterResourceGroupID(ctx context.Context, partitionKey, resourceGroupID string) (*api.OpenShiftClusterDocuments, error)
}

// NewOpenShiftClusters returns a new OpenShiftClusters
func NewOpenShiftClusters(ctx context.Context, uuid string, dbc cosmosdb.DatabaseClient, dbid, collid string) (OpenShiftClusters, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbid)

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

	triggerc := cosmosdb.NewTriggerClient(collc, collid)
	for _, trigger := range triggers {
		_, err := triggerc.Create(ctx, trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	return &openShiftClusters{
		c:     cosmosdb.NewOpenShiftClusterDocumentClient(collc, collid),
		collc: collc,
		uuid:  uuid,
	}, nil
}

func (c *openShiftClusters) Create(ctx context.Context, doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	if doc.Key != strings.ToLower(doc.Key) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	var err error
	doc.PartitionKey, err = c.partitionKey(doc.Key)
	if err != nil {
		return nil, err
	}

	doc, err = c.c.Create(ctx, doc.PartitionKey, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *openShiftClusters) Get(ctx context.Context, key string) (*api.OpenShiftClusterDocument, error) {
	if key != strings.ToLower(key) {
		return nil, fmt.Errorf("key %q is not lower case", key)
	}

	partitionKey, err := c.partitionKey(key)
	if err != nil {
		return nil, err
	}

	docs, err := c.c.QueryAll(ctx, partitionKey, &cosmosdb.Query{
		Query: "SELECT * FROM OpenShiftClusters doc WHERE doc.key = @key",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@key",
				Value: key,
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	switch {
	case len(docs.OpenShiftClusterDocuments) > 1:
		return nil, fmt.Errorf("read %d documents, expected <= 1", len(docs.OpenShiftClusterDocuments))
	case len(docs.OpenShiftClusterDocuments) == 1:
		return docs.OpenShiftClusterDocuments[0], nil
	default:
		return nil, &cosmosdb.Error{StatusCode: http.StatusNotFound}
	}
}

// QueueLength returns OpenShiftClusters un-queued document count.
// If error occurs, 0 is returned with error message
func (c *openShiftClusters) QueueLength(ctx context.Context, collid string) (int, error) {
	partitions, err := c.collc.PartitionKeyRanges(ctx, collid)
	if err != nil {
		return 0, err
	}

	var countTotal int
	for _, r := range partitions.PartitionKeyRanges {
		result := c.c.Query("", &cosmosdb.Query{
			Query: `SELECT VALUE COUNT(1) FROM OpenShiftClusters doc WHERE doc.openShiftCluster.properties.provisioningState IN ("Creating", "Deleting", "Updating", "AdminUpdating") AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
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

func (c *openShiftClusters) Patch(ctx context.Context, key string, f func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	return c.patch(ctx, key, f, nil)
}

func (c *openShiftClusters) patch(ctx context.Context, key string, f func(*api.OpenShiftClusterDocument) error, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	var doc *api.OpenShiftClusterDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, key)
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

func (c *openShiftClusters) PatchWithLease(ctx context.Context, key string, f func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	return c.patchWithLease(ctx, key, f, nil)
}

func (c *openShiftClusters) patchWithLease(ctx context.Context, key string, f func(*api.OpenShiftClusterDocument) error, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	return c.patch(ctx, key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, options)
}

func (c *openShiftClusters) Update(ctx context.Context, doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	return c.update(ctx, doc, nil)
}

func (c *openShiftClusters) update(ctx context.Context, doc *api.OpenShiftClusterDocument, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	if doc.Key != strings.ToLower(doc.Key) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Replace(ctx, doc.PartitionKey, doc, options)
}

func (c *openShiftClusters) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	if doc.Key != strings.ToLower(doc.Key) {
		return fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Delete(ctx, doc.PartitionKey, doc, &cosmosdb.Options{NoETag: true})
}

func (c *openShiftClusters) ChangeFeed() cosmosdb.OpenShiftClusterDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *openShiftClusters) List(continuation string) cosmosdb.OpenShiftClusterDocumentIterator {
	return c.c.List(&cosmosdb.Options{Continuation: continuation})
}

func (c *openShiftClusters) ListByPrefix(subscriptionID, prefix, continuation string) (cosmosdb.OpenShiftClusterDocumentIterator, error) {
	if prefix != strings.ToLower(prefix) {
		return nil, fmt.Errorf("prefix %q is not lower case", prefix)
	}

	return c.c.Query(
		subscriptionID,
		&cosmosdb.Query{
			Query: "SELECT * FROM OpenShiftClusters doc WHERE STARTSWITH(doc.key, @prefix)",
			Parameters: []cosmosdb.Parameter{
				{
					Name:  "@prefix",
					Value: prefix,
				},
			},
		},
		&cosmosdb.Options{Continuation: continuation},
	), nil
}

func (c *openShiftClusters) Dequeue(ctx context.Context) (*api.OpenShiftClusterDocument, error) {
	i := c.c.Query("", &cosmosdb.Query{
		Query: `SELECT * FROM OpenShiftClusters doc WHERE doc.openShiftCluster.properties.provisioningState IN ("Creating", "Deleting", "Updating", "AdminUpdating") AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
	}, nil)

	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return nil, err
		}
		if docs == nil {
			return nil, nil
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
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

func (c *openShiftClusters) Lease(ctx context.Context, key string) (*api.OpenShiftClusterDocument, error) {
	return c.patchWithLease(ctx, key, func(doc *api.OpenShiftClusterDocument) error {
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *openShiftClusters) EndLease(ctx context.Context, key string, provisioningState, failedProvisioningState api.ProvisioningState, adminUpdateError *string) (*api.OpenShiftClusterDocument, error) {
	return c.patchWithLease(ctx, key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisioningState = provisioningState
		doc.OpenShiftCluster.Properties.FailedProvisioningState = failedProvisioningState

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		if provisioningState != api.ProvisioningStateFailed {
			doc.Dequeues = 0
		}
		// If EndLease is called while cluster is still in terminal phase,
		// we clean AsyncOperationID. Otherwise it just handover between backends.
		if provisioningState.IsTerminal() {
			if adminUpdateError != nil {
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = *adminUpdateError
			}

			doc.CorrelationData = nil
			doc.OpenShiftCluster.Properties.LastProvisioningState = ""
			doc.AsyncOperationID = ""
		}

		return nil
	}, nil)
}

func (c *openShiftClusters) partitionKey(key string) (string, error) {
	r, err := azure.ParseResourceID(key)
	return r.SubscriptionID, err
}

func (c *openShiftClusters) GetByClientID(ctx context.Context, partitionKey, clientID string) (*api.OpenShiftClusterDocuments, error) {
	docs, err := c.c.QueryAll(ctx, partitionKey, &cosmosdb.Query{
		Query: "SELECT * FROM OpenShiftClusters doc WHERE doc.clientIdKey = @clientID",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@clientID",
				Value: clientID,
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (c *openShiftClusters) GetByClusterResourceGroupID(ctx context.Context, partitionKey, resourceGroupID string) (*api.OpenShiftClusterDocuments, error) {
	docs, err := c.c.QueryAll(ctx, partitionKey, &cosmosdb.Query{
		Query: "SELECT * FROM OpenShiftClusters doc WHERE doc.clusterResourceGroupIdKey = @resourceGroupID",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@resourceGroupID",
				Value: resourceGroupID,
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	return docs, nil
}
