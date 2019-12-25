package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type openShiftClusters struct {
	c    cosmosdb.OpenShiftClusterDocumentClient
	uuid string
}

// OpenShiftClusters is the database interface for OpenShiftClusterDocuments
type OpenShiftClusters interface {
	Create(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Get(string) (*api.OpenShiftClusterDocument, error)
	Patch(string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	Update(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Delete(*api.OpenShiftClusterDocument) error
	ListByPrefix(string, string) (cosmosdb.OpenShiftClusterDocumentIterator, error)
	Dequeue() (*api.OpenShiftClusterDocument, error)
	Lease(string) (*api.OpenShiftClusterDocument, error)
	EndLease(string, api.ProvisioningState, api.ProvisioningState) (*api.OpenShiftClusterDocument, error)
}

// NewOpenShiftClusters returns a new OpenShiftClusters
func NewOpenShiftClusters(uuid string, dbc cosmosdb.DatabaseClient, dbid, collid string) (OpenShiftClusters, error) {
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
		_, err := triggerc.Create(trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	return &openShiftClusters{
		c:    cosmosdb.NewOpenShiftClusterDocumentClient(collc, collid),
		uuid: uuid,
	}, nil
}

func (c *openShiftClusters) Create(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	if doc.Key != strings.ToLower(doc.Key) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	var err error
	doc.PartitionKey, err = c.partitionKey(doc.Key)
	if err != nil {
		return nil, err
	}

	doc, err = c.c.Create(doc.PartitionKey, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *openShiftClusters) Get(key string) (*api.OpenShiftClusterDocument, error) {
	if key != strings.ToLower(key) {
		return nil, fmt.Errorf("key %q is not lower case", key)
	}

	partitionKey, err := c.partitionKey(key)
	if err != nil {
		return nil, err
	}

	docs, err := c.c.QueryAll(partitionKey, &cosmosdb.Query{
		Query: "SELECT * FROM OpenShiftClusters doc WHERE doc.key = @key",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@key",
				Value: key,
			},
		},
	})
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

func (c *openShiftClusters) Patch(key string, f func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	return c.patch(key, f, nil)
}

func (c *openShiftClusters) patch(key string, f func(*api.OpenShiftClusterDocument) error, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	var doc *api.OpenShiftClusterDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(key)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.update(doc, options)
		return
	})

	return doc, err
}

func (c *openShiftClusters) Update(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	return c.update(doc, nil)
}

func (c *openShiftClusters) update(doc *api.OpenShiftClusterDocument, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	if doc.Key != strings.ToLower(doc.Key) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Replace(doc.PartitionKey, doc, options)
}

func (c *openShiftClusters) Delete(doc *api.OpenShiftClusterDocument) error {
	if doc.Key != strings.ToLower(doc.Key) {
		return fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Delete(doc.PartitionKey, doc, &cosmosdb.Options{NoETag: true})
}

func (c *openShiftClusters) ListByPrefix(subscriptionID string, prefix string) (cosmosdb.OpenShiftClusterDocumentIterator, error) {
	if prefix != strings.ToLower(prefix) {
		return nil, fmt.Errorf("prefix %q is not lower case", prefix)
	}

	return c.c.Query(subscriptionID, &cosmosdb.Query{
		Query: "SELECT * FROM OpenShiftClusters doc WHERE STARTSWITH(doc.key, @prefix)",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@prefix",
				Value: prefix,
			},
		},
	}), nil
}

func (c *openShiftClusters) Dequeue() (*api.OpenShiftClusterDocument, error) {
	i := c.c.Query("", &cosmosdb.Query{
		Query: `SELECT * FROM OpenShiftClusters doc WHERE NOT (doc.openShiftCluster.properties.provisioningState IN ("Succeeded", "Failed")) AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
	})

	for {
		docs, err := i.Next()
		if err != nil {
			return nil, err
		}
		if docs == nil {
			return nil, nil
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			doc.LeaseOwner = c.uuid
			doc.Dequeues++
			doc, err = c.update(doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}
}

func (c *openShiftClusters) Lease(key string) (*api.OpenShiftClusterDocument, error) {
	return c.patch(key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *openShiftClusters) EndLease(key string, provisioningState, failedProvisioningState api.ProvisioningState) (*api.OpenShiftClusterDocument, error) {
	return c.patch(key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		doc.OpenShiftCluster.Properties.ProvisioningState = provisioningState
		doc.OpenShiftCluster.Properties.FailedProvisioningState = failedProvisioningState

		doc.LeaseOwner = ""
		doc.LeaseExpires = 0

		if provisioningState == api.ProvisioningStateSucceeded {
			doc.Dequeues = 0
		}

		doc.AsyncOperationID = ""

		return nil
	}, nil)
}

func (c *openShiftClusters) partitionKey(key string) (string, error) {
	r, err := azure.ParseResourceID(key)
	return r.SubscriptionID, err
}
