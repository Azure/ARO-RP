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
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	ClusterManagerConfigurationsGetQuery = `SELECT * FROM ClusterManagerConfigurations doc WHERE doc.key = @key`
)

type clusterManagerConfiguration struct {
	c             cosmosdb.ClusterManagerConfigurationDocumentClient
	collc         cosmosdb.CollectionClient
	uuid          string
	uuidGenerator uuid.Generator
}

type ClusterManagerConfigurations interface {
	Create(context.Context, *api.ClusterManagerConfigurationDocument) (*api.ClusterManagerConfigurationDocument, error)
	Get(context.Context, string) (*api.ClusterManagerConfigurationDocument, error)
	Update(context.Context, *api.ClusterManagerConfigurationDocument) (*api.ClusterManagerConfigurationDocument, error)
	Delete(context.Context, *api.ClusterManagerConfigurationDocument) error
	ChangeFeed() cosmosdb.ClusterManagerConfigurationDocumentIterator
	NewUUID() string
}

func NewClusterManagerConfigurations(ctx context.Context, isDevelopmentMode bool, dbc cosmosdb.DatabaseClient) (ClusterManagerConfigurations, error) {
	dbid, err := Name(isDevelopmentMode)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	documentClient := cosmosdb.NewClusterManagerConfigurationDocumentClient(collc, collClusterManager)
	return NewClusterManagerConfigurationsWithProvidedClient(documentClient, collc, uuid.DefaultGenerator.Generate(), uuid.DefaultGenerator), nil
}

func NewClusterManagerConfigurationsWithProvidedClient(client cosmosdb.ClusterManagerConfigurationDocumentClient, collectionClient cosmosdb.CollectionClient, uuid string, uuidGenerator uuid.Generator) ClusterManagerConfigurations {
	return &clusterManagerConfiguration{
		c:             client,
		collc:         collectionClient,
		uuid:          uuid,
		uuidGenerator: uuidGenerator,
	}
}

func (c *clusterManagerConfiguration) NewUUID() string {
	return c.uuidGenerator.Generate()
}

func (c *clusterManagerConfiguration) Create(ctx context.Context, doc *api.ClusterManagerConfigurationDocument) (*api.ClusterManagerConfigurationDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
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

func (c *clusterManagerConfiguration) Get(ctx context.Context, id string) (*api.ClusterManagerConfigurationDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}
	partitionKey, err := c.partitionKey(id)
	if err != nil {
		return nil, err
	}

	docs, err := c.c.QueryAll(ctx, partitionKey, &cosmosdb.Query{
		Query: ClusterManagerConfigurationsGetQuery,
		Parameters: []cosmosdb.Parameter{
			{
				Name:  "@key",
				Value: id,
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	switch {
	case len(docs.ClusterManagerConfigurationDocuments) > 1:
		return nil, fmt.Errorf("read %d documents, expected <= 1", len(docs.ClusterManagerConfigurationDocuments))
	case len(docs.ClusterManagerConfigurationDocuments) == 1:
		return docs.ClusterManagerConfigurationDocuments[0], nil
	default:
		return nil, &cosmosdb.Error{StatusCode: http.StatusNotFound}
	}

}

func (c *clusterManagerConfiguration) Update(ctx context.Context, doc *api.ClusterManagerConfigurationDocument) (*api.ClusterManagerConfigurationDocument, error) {
	return c.update(ctx, doc, nil)
}

func (c *clusterManagerConfiguration) update(ctx context.Context, doc *api.ClusterManagerConfigurationDocument, options *cosmosdb.Options) (*api.ClusterManagerConfigurationDocument, error) {
	if doc.Key != strings.ToLower(doc.Key) {
		return nil, fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Replace(ctx, doc.PartitionKey, doc, options)
}

func (c *clusterManagerConfiguration) Delete(ctx context.Context, doc *api.ClusterManagerConfigurationDocument) error {
	if doc.ID != strings.ToLower(doc.ID) {
		return fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Delete(ctx, doc.PartitionKey, doc, &cosmosdb.Options{NoETag: true})
}

func (c *clusterManagerConfiguration) ChangeFeed() cosmosdb.ClusterManagerConfigurationDocumentIterator {
	return c.c.ChangeFeed(nil)
}

func (c *clusterManagerConfiguration) partitionKey(key string) (string, error) {
	r, err := azure.ParseResourceID(key)
	return r.SubscriptionID, err
}
