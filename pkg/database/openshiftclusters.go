package database

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	"github.com/jim-minter/rp/pkg/env"
)

type openShiftClusters struct {
	c    cosmosdb.OpenShiftClusterDocumentClient
	uuid uuid.UUID
}

// OpenShiftClusters is the database interface for OpenShiftClusterDocuments
type OpenShiftClusters interface {
	Create(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Get(string) (*api.OpenShiftClusterDocument, error)
	Patch(string, func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error)
	Update(*api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error)
	Delete(string) error
	ListByPrefix(string, string) cosmosdb.OpenShiftClusterDocumentIterator
	Dequeue() (*api.OpenShiftClusterDocument, error)
	Lease(string) (*api.OpenShiftClusterDocument, error)
}

// NewOpenShiftClusters returns a new OpenShiftClusters
func NewOpenShiftClusters(ctx context.Context, env env.Interface, uuid uuid.UUID, dbid, collid string) (OpenShiftClusters, error) {
	databaseAccount, masterKey, err := env.CosmosDB(ctx)
	if err != nil {
		return nil, err
	}

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, databaseAccount, masterKey)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	triggerc := cosmosdb.NewTriggerClient(collc, collid)
	_, err = triggerc.Create(&cosmosdb.Trigger{
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
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return nil, err
	}

	return &openShiftClusters{
		c:    cosmosdb.NewOpenShiftClusterDocumentClient(collc, collid),
		uuid: uuid,
	}, nil
}

func (c *openShiftClusters) Create(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	doc.PartitionKey = r.SubscriptionID
	doc.OpenShiftCluster.ID = strings.ToLower(doc.OpenShiftCluster.ID)
	doc.OpenShiftCluster.Name = strings.ToLower(doc.OpenShiftCluster.Name)
	doc.OpenShiftCluster.Type = strings.ToLower(doc.OpenShiftCluster.Type)

	doc, err = c.c.Create(doc.PartitionKey, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *openShiftClusters) Get(resourceID string) (*api.OpenShiftClusterDocument, error) {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	docs, err := c.c.QueryAll(r.SubscriptionID, &cosmosdb.Query{
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
		return nil, &cosmosdb.Error{StatusCode: http.StatusNotFound}
	}
}

func (c *openShiftClusters) Patch(resourceID string, f func(*api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
	return c.patch(resourceID, f, nil)
}

func (c *openShiftClusters) patch(resourceID string, f func(*api.OpenShiftClusterDocument) error, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
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

		doc, err = c.update(doc, options)
		return
	})

	return doc, err
}

func (c *openShiftClusters) Update(doc *api.OpenShiftClusterDocument) (*api.OpenShiftClusterDocument, error) {
	return c.update(doc, nil)
}

func (c *openShiftClusters) update(doc *api.OpenShiftClusterDocument, options *cosmosdb.Options) (*api.OpenShiftClusterDocument, error) {
	doc.OpenShiftCluster.ID = strings.ToLower(doc.OpenShiftCluster.ID)
	doc.OpenShiftCluster.Name = strings.ToLower(doc.OpenShiftCluster.Name)
	doc.OpenShiftCluster.Type = strings.ToLower(doc.OpenShiftCluster.Type)

	return c.c.Replace(doc.PartitionKey, doc, options)
}

func (c *openShiftClusters) Delete(resourceID string) error {
	return cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err := c.Get(resourceID)
		if err != nil {
			return
		}

		return c.c.Delete(doc.PartitionKey, doc, nil)
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

func (c *openShiftClusters) Dequeue() (*api.OpenShiftClusterDocument, error) {
	i := c.c.Query("", &cosmosdb.Query{
		Query: `SELECT * FROM OpenShiftClusterDocuments doc WHERE NOT (doc.openShiftCluster.properties.provisioningState IN ("Succeeded", "Failed")) AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
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
			doc.LeaseOwner = &c.uuid
			doc.Dequeues++
			doc, err = c.update(doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
			if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
				continue
			}
			return doc, err
		}
	}
}

func (c *openShiftClusters) Lease(resourceID string) (*api.OpenShiftClusterDocument, error) {
	return c.patch(resourceID, func(doc *api.OpenShiftClusterDocument) error {
		if doc.LeaseOwner == nil || !uuid.Equal(*doc.LeaseOwner, c.uuid) {
			return fmt.Errorf("lost lease")
		}
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}
