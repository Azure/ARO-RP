package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

// controller updates the monitor document with the list of buckets balanced between
// registered workers
func (s *service) controller(ctx context.Context) error {
	var doc *api.BucketServiceDocument
	var err error

	// if we know we're not the controller, attempt to gain the lease on the monitor
	// document
	if !s.isController {
		doc, err = s.dbBucketServices.TryLease(ctx, s.serviceName)
		if err != nil || doc == nil {
			return err
		}
		s.isController = true
	}

	// we know we're not the controller; give up
	if !s.isController {
		return nil
	}

	// we think we're the controller.  Gather up all the registered workers
	// including ourself, balance buckets between them and write the bucket
	// allocations to the database.  If it turns out that we're not the controller,
	// the patch will fail
	_, err = s.dbBucketServices.PatchWithLease(ctx, doc.ID, func(doc *api.BucketServiceDocument) error {
		docs, err := s.dbBucketServices.ListBucketServices(ctx, s.serviceName)
		if err != nil {
			return err
		}

		var workers []string
		if docs != nil {
			workers = make([]string, 0, len(docs.BucketServiceDocuments))
			for _, doc := range docs.BucketServiceDocuments {
				workers = append(workers, doc.ID)
			}
		}

		doc.Buckets = s.b.Balance(workers, doc.Buckets)

		return nil
	})
	if err != nil && err.Error() == "lost lease" {
		s.isController = false
	}
	return err
}
