package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

type clusterManagerConfigurationBackend struct {
	*backend

	configIterator cosmosdb.ClusterManagerConfigurationDocumentIterator
	jsonSerializer *kjson.Serializer
}

func newClusterManagerConfigurationBackend(b *backend) *clusterManagerConfigurationBackend {
	return &clusterManagerConfigurationBackend{
		backend:        b,
		configIterator: nil, // created as needed
		jsonSerializer: kjson.NewSerializerWithOptions(
			kjson.DefaultMetaFactory,
			scheme.Scheme,
			scheme.Scheme,
			kjson.SerializerOptions{}),
	}
}

func (cmc *clusterManagerConfigurationBackend) try(ctx context.Context, backendDoc *api.BackendDocument) (bool, error) {
	log := cmc.backend.baseLog

	if backendDoc == nil {
		cmc.configIterator = nil
		return false, nil
	}

	if cmc.configIterator == nil {
		var options *cosmosdb.Options
		if backendDoc.ClusterManagerConfigurations != nil {
			// Create a change feed iterator starting from where the last
			// lease owner left off, as recorded in the Continuation value.
			options = &cosmosdb.Options{
				Continuation: backendDoc.ClusterManagerConfigurations.Continuation,
			}
		}
		// Keep this around for as long as we retain the backend lease.
		cmc.configIterator = cmc.dbClusterManagerConfigurations.ChangeFeed(options)
	}

	restConfig, err := cmc.backend.env.LiveConfig().HiveRestConfig(ctx, 1)
	if err != nil {
		log.Info(err) // TODO(hive): Update to fail once we have Hive everywhere in prod and dev
		return false, nil
	}

	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return false, err
	}

	docs, err := cmc.configIterator.Next(ctx, -1)
	if err != nil {
		return false, err
	}

	workWasDone := false

	if docs != nil {
		for _, doc := range docs.ClusterManagerConfigurationDocuments {
			encodedResource := []byte(doc.GetResources())
			object, gvk, err := cmc.jsonSerializer.Decode(encodedResource, nil, nil)
			if err != nil {
				log.Errorf("Error decoding document ID %q: %s", doc.ID, err)
			} else if doc.Deleting {
				gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
				acc, err := meta.Accessor(object)
				if err != nil {
					return false, err
				}
				err = dh.EnsureDeleted(ctx, gk.String(), acc.GetNamespace(), acc.GetName())
				if err != nil {
					return false, err
				}
			} else {
				err = dh.Ensure(ctx, object)
				if err != nil {
					return false, err
				}
			}
		}

		// Patch the backend document with the new continuation value.
		// If it turns out we lost the lease then the patch will fail
		// and the next lease owner may repeat these changes. But the
		// dynamichelper methods are idempotent so it should be fine.
		_, err := cmc.backend.patchWithLease(ctx, func(doc *api.BackendDocument) error {
			if doc.ClusterManagerConfigurations == nil {
				doc.ClusterManagerConfigurations = &api.ClusterManagerConfigurationsBackend{}
			}
			doc.ClusterManagerConfigurations.Continuation = cmc.configIterator.Continuation()
			return nil
		})
		if err != nil {
			return false, err
		}

		workWasDone = true
	}

	return workWasDone, nil
}
