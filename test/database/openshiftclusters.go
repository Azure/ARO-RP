package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func (c *fakeOpenShiftClusters) Query(name string, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	c.lock.Lock()
	defer c.lock.Unlock()

	results := make([]*api.OpenShiftClusterDocument, 0)

	switch query.Query {
	case database.OpenShiftClustersDequeueQuery:
		for _, k := range c.docs {
			var include bool

			r, err := c.fromString(k)
			if err != nil {
				return nil
			}

			switch r.OpenShiftCluster.Properties.ProvisioningState {
			case
				api.ProvisioningStateCreating,
				api.ProvisioningStateUpdating,
				api.ProvisioningStateAdminUpdating,
				api.ProvisioningStateDeleting:
				include = true
			}

			if include && (r.LeaseExpires > 0 && int64(r.LeaseExpires) < time.Now().Unix()) {
				include = false
			}

			if include {
				results = append(results, r)
			}
		}
	}

	return &fakeOpenShiftClustersRawIterator{
		docs: results,
	}
}

func (c *fakeOpenShiftClusters) QueryAll(ctx context.Context, partitionkey string, query *cosmosdb.Query, options *cosmosdb.Options) (*api.OpenShiftClusterDocuments, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	results := make([]*api.OpenShiftClusterDocument, 0)

	switch {
	case len(query.Parameters) == 1 && query.Parameters[0].Name == "@key":
		for _, k := range c.docs {
			r, err := c.fromString(k)
			if err != nil {
				return nil, err
			}

			if r.Key == query.Parameters[0].Value {
				results = append(results, r)
			}
		}

	default:
		return nil, errors.New("query not implemented")
	}

	return &api.OpenShiftClusterDocuments{
		Count:                     len(results),
		OpenShiftClusterDocuments: results,
	}, nil
}
