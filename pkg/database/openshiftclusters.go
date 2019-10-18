package database

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	"github.com/jim-minter/rp/pkg/util/resource"
)

type openShiftClusters struct {
	c cosmosdb.OpenShiftClusterDocumentClient
}

// OpenShiftClusters is the database interface for OpenShiftClusterDocuments
type OpenShiftClusters interface {
	Create(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Get(string) (*api.OpenShiftClusterDocument, error)
	Patch(string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	Update(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Delete(string) error
	ListUnqueued() cosmosdb.OpenShiftClusterDocumentIterator
	ListByPrefix(string, string) cosmosdb.OpenShiftClusterDocumentIterator
}

// NewOpenShiftClusters returns a new OpenShiftClusters
func NewOpenShiftClusters(dbc cosmosdb.DatabaseClient, dbid, collid string) OpenShiftClusters {
	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	return &openShiftClusters{
		c: cosmosdb.NewOpenShiftClusterDocumentClient(collc, collid),
	}
}

func (c *openShiftClusters) Create(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	var err error
	doc.SubscriptionID, err = resource.SubscriptionID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	doc.OpenShiftCluster.ID = strings.ToLower(doc.OpenShiftCluster.ID)
	doc.OpenShiftCluster.Name = strings.ToLower(doc.OpenShiftCluster.Name)
	doc.OpenShiftCluster.Type = strings.ToLower(doc.OpenShiftCluster.Type)

	doc, err = c.c.Create(doc.SubscriptionID, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *openShiftClusters) Get(resourceID string) (*api.OpenShiftClusterDocument, error) {
	subscriptionID, err := resource.SubscriptionID(resourceID)
	if err != nil {
		return nil, err
	}

	docs, err := c.c.QueryAll(subscriptionID, &cosmosdb.Query{
		Query: "SELECT * FROM OpenshiftClusterDocuments doc WHERE doc.openShiftCluster.id = @id",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@id",
				Value: strings.ToLower(resourceID),
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
		return nil, nil
	}
}

func (c *openShiftClusters) Patch(resourceID string, f func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	var doc *api.OpenShiftClusterDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(resourceID)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.Update(doc)
		return
	})

	return doc, err
}

func (c *openShiftClusters) Update(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	doc.OpenShiftCluster.ID = strings.ToLower(doc.OpenShiftCluster.ID)
	doc.OpenShiftCluster.Name = strings.ToLower(doc.OpenShiftCluster.Name)
	doc.OpenShiftCluster.Type = strings.ToLower(doc.OpenShiftCluster.Type)

	return c.c.Replace(doc.SubscriptionID, doc, nil)
}

func (c *openShiftClusters) Delete(resourceID string) error {
	return cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err := c.Get(resourceID)
		if err != nil {
			return
		}

		return c.c.Delete(doc.SubscriptionID, doc, nil)
	})
}

func (c *openShiftClusters) ListUnqueued() cosmosdb.OpenShiftClusterDocumentIterator {
	return c.c.Query("", &cosmosdb.Query{
		Query: "SELECT * FROM OpenshiftClusterDocuments doc WHERE doc.unqueued = true",
	})
}

func (c *openShiftClusters) ListByPrefix(subscriptionID, prefix string) cosmosdb.OpenShiftClusterDocumentIterator {
	return c.c.Query(subscriptionID, &cosmosdb.Query{
		Query: "SELECT * FROM OpenshiftClusterDocuments doc WHERE STARTSWITH(doc.openShiftCluster.id, @prefix)",
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@prefix",
				Value: strings.ToLower(prefix),
			},
		},
	})
}
