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

type bucketServices struct {
	c    cosmosdb.BucketServiceDocumentClient
	uuid string
}

// BucketServices is the database interface for BucketServiceDocuments
type BucketServices interface {
	Create(context.Context, *api.BucketServiceDocument) (*api.BucketServiceDocument, error)
	PatchWithLease(context.Context, string, func(*api.BucketServiceDocument) error) (*api.BucketServiceDocument, error)
	TryLease(ctx context.Context, service string) (*api.BucketServiceDocument, error)
	ListBuckets(ctx context.Context, service string) ([]int, error)
	ListBucketServices(ctx context.Context, service string) (*api.BucketServiceDocuments, error)
	BucketServiceHeartbeat(ctx context.Context, service string) error
}

// NewBucketServices returns a new BucketServices
func NewBucketServices(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string) (BucketServices, error) {
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

	triggerc := cosmosdb.NewTriggerClient(collc, collBucketServices)
	for _, trigger := range triggers {
		_, err := triggerc.Create(ctx, trigger)
		if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
			return nil, err
		}
	}

	return &bucketServices{
		c:    cosmosdb.NewBucketServiceDocumentClient(collc, collBucketServices),
		uuid: uuid.DefaultGenerator.Generate(),
	}, nil
}

func (c *bucketServices) Create(ctx context.Context, doc *api.BucketServiceDocument) (*api.BucketServiceDocument, error) {
	doc, err := c.c.Create(ctx, doc.ServiceName, doc, nil)

	if err, ok := err.(*cosmosdb.Error); ok && err.StatusCode == http.StatusConflict {
		err.StatusCode = http.StatusPreconditionFailed
	}

	return doc, err
}

func (c *bucketServices) get(ctx context.Context, id string) (*api.BucketServiceDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, "", id, nil)
}

func (c *bucketServices) patch(ctx context.Context, id string, f func(*api.BucketServiceDocument) error, options *cosmosdb.Options) (*api.BucketServiceDocument, error) {
	var doc *api.BucketServiceDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.get(ctx, id)
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

func (c *bucketServices) PatchWithLease(ctx context.Context, id string, f func(*api.BucketServiceDocument) error) (*api.BucketServiceDocument, error) {
	return c.patch(ctx, id, func(doc *api.BucketServiceDocument) error {
		if doc.LeaseOwner != c.uuid {
			return fmt.Errorf("lost lease")
		}

		return f(doc)
	}, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
}

func (c *bucketServices) update(ctx context.Context, doc *api.BucketServiceDocument, options *cosmosdb.Options) (*api.BucketServiceDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Replace(ctx, doc.ID, doc, options)
}

func (c *bucketServices) TryLease(ctx context.Context, service string) (*api.BucketServiceDocument, error) {
	if service != strings.ToLower(service) {
		return nil, fmt.Errorf("id %q is not lower case", service)
	}

	doc, err := c.GetServiceController(ctx, service)
	if err != nil {
		return nil, err
	}
	doc.LeaseOwner = c.uuid
	doc, err = c.update(ctx, doc, &cosmosdb.Options{PreTriggers: []string{"renewLease"}})
	if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) { // someone else got there first
		return nil, err
	}
	return doc, err
}

func (c *bucketServices) GetServiceController(ctx context.Context, service string) (*api.BucketServiceDocument, error) {
	if service != strings.ToLower(service) {
		return nil, fmt.Errorf("service %q is not lower case", service)
	}
	r, err := c.c.QueryAll(ctx, service, &cosmosdb.Query{
		Query: `SELECT * FROM BucketServices doc WHERE doc.serviceName = @serviceName AND doc.serviceRole = "controller" AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "serviceName",
				Value: service,
			},
		},
	}, nil)

	if err != nil {
		return nil, err
	}
	if len(r.BucketServiceDocuments) == 0 {
		return nil, fmt.Errorf("no buckets found for %v", service)
	}

	return r.BucketServiceDocuments[0], nil
}

func (c *bucketServices) ListBucketServices(ctx context.Context, service string) (*api.BucketServiceDocuments, error) {
	return c.c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: `SELECT * FROM BucketServices doc WHERE doc.serviceRole != "controller" AND doc.serviceName != "@name`,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@name",
				Value: service,
			},
		},
	}, nil)
}

func (c *bucketServices) ListBuckets(ctx context.Context, service string) (buckets []int, err error) {
	if service != strings.ToLower(service) {
		return nil, fmt.Errorf("id %q is not lower case", service)
	}

	doc, err := c.GetServiceController(ctx, service)
	if err != nil || doc == nil {
		return nil, err
	}

	for i, BucketService := range doc.Buckets {
		if BucketService == c.uuid {
			buckets = append(buckets, i)
		}
	}

	return buckets, nil
}

func (c *bucketServices) BucketServiceHeartbeat(ctx context.Context, service string) error {
	doc := &api.BucketServiceDocument{
		ID:          c.uuid,
		TTL:         60,
		ServiceName: service,
		ServiceRole: c.uuid,
	}
	_, err := c.update(ctx, doc, &cosmosdb.Options{NoETag: true})
	if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		_, err = c.Create(ctx, doc)
	}
	return err
}
