package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func (c *fakeOpenShiftClusters) Query(name string, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	c.lock.Lock()
	defer c.lock.Unlock()

	if query.Query == database.OpenShiftClustersDequeueQuery {

		results := make([]*api.OpenShiftClusterDocument, 0)

		for _, k := range c.docs {

			include := false

			var res map[interface{}]interface{}
			d := codec.NewDecoder(bytes.NewBufferString(*k), c.jsonHandle)
			err := d.Decode(&res)
			if err != nil {
				return nil
			}

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

		return &fakeOpenShiftClustersRawIterator{
			docs: results,
		}

	}

	return nil
}
func (c *fakeOpenShiftClusters) QueryAll(ctx context.Context, partitionkey string, query *cosmosdb.Query, options *cosmosdb.Options) (*api.OpenShiftClusterDocuments, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	results := make([]*api.OpenShiftClusterDocument, 0)

	if len(query.Parameters) != 1 || query.Parameters[0].Name != "@key" {
		return nil, errors.New("not implemented but for get")
	}

	for _, k := range c.docs {

		var res map[interface{}]interface{}
		d := codec.NewDecoder(bytes.NewBufferString(*k), c.jsonHandle)
		err := d.Decode(&res)
		if err != nil {
			return nil, err
		}

		r, err := c.fromString(k)
		if err != nil {
			return nil, err
		}

		if res["key"] == query.Parameters[0].Value {
			results = append(results, r)
		}
	}

	return &api.OpenShiftClusterDocuments{
		Count:                     len(results),
		OpenShiftClusterDocuments: results,
	}, nil

}
