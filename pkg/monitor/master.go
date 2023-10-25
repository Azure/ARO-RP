package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
)

// master updates the monitor document with the list of buckets balanced between
// registered monitors
func (mon *monitor) master(ctx context.Context) error {
	var doc *api.MonitorDocument
	var err error

	// if we know we're not the master, attempt to gain the lease on the monitor
	// document
	if !mon.isMaster {
		doc, err := mon.dbMonitors.TryLease(ctx)
		if err != nil || doc == nil {
			return err
		}
		mon.isMaster = true
	}

	// we know we're not the master; give up
	if !mon.isMaster {
		return nil
	}

	// we think we're the master.  Gather up all the registered monitors
	// including ourself, balance buckets between them and write the bucket
	// allocations to the database.  If it turns out that we're not the master,
	// the patch will fail
	_, err = mon.dbMonitors.PatchWithLease(ctx, "master", func(doc *api.MonitorDocument) error {
		docs, err := mon.dbMonitors.ListMonitors(ctx)
		if err != nil {
			return err
		}

		var monitors []string
		if docs != nil {
			monitors = make([]string, 0, len(docs.MonitorDocuments))
			for _, doc := range docs.MonitorDocuments {
				monitors = append(monitors, doc.ID)
			}
		}

		mon.b.Balance(monitors, doc)

		return nil
	})
	if err != nil && err.Error() == "lost lease" {
		mon.isMaster = false
	}
	return err
}
