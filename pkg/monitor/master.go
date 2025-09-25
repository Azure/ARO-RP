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
	dbMonitors, err := mon.dbGroup.Monitors()
	if err != nil {
		return err
	}

	// if we know we're not the master, attempt to gain the lease on the monitor
	// document
	if !mon.isMaster {
		doc, err := dbMonitors.TryLease(ctx)
		if err != nil || doc == nil {
			return err
		}
		mon.isMaster = true
	}

	// we know we're not the master; give up
	if !mon.isMaster {
		return nil
	}
	mon.baseLog.Info("I Am master")

	// we think we're the master.  Gather up all the registered monitors
	// including ourself, balance buckets between them and write the bucket
	// allocations to the database.  If it turns out that we're not the master,
	// the patch will fail
	_, err = dbMonitors.PatchWithLease(ctx, "master", func(doc *api.MonitorDocument) error {
		docs, err := dbMonitors.ListMonitors(ctx)
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

		mon.baseLog.Info("Balancing buckets across monitors")
		mon.balance(monitors, doc)

		return nil
	})
	if err != nil && err.Error() == "lost lease" {
		mon.isMaster = false
	}
	return err
}

// balance shares out buckets over a slice of registered monitors
func (mon *monitor) balance(monitors []string, doc *api.MonitorDocument) {
	// initialise doc.Monitor
	if doc.Monitor == nil {
		doc.Monitor = &api.Monitor{}
	}

	// ensure len(doc.Monitor.Buckets) == mon.bucketCount: this should only do
	// anything on the very first run
	if len(doc.Monitor.Buckets) < mon.bucketCount {
		doc.Monitor.Buckets = append(doc.Monitor.Buckets, make([]string, mon.bucketCount-len(doc.Monitor.Buckets))...)
	}
	if len(doc.Monitor.Buckets) > mon.bucketCount { // should never happen
		doc.Monitor.Buckets = doc.Monitor.Buckets[:mon.bucketCount]
	}

	var unallocated []int
	m := make(map[string][]int, len(monitors)) // map of monitor to list of buckets it owns
	for _, monitor := range monitors {
		m[monitor] = nil
	}

	var target int // target number of buckets per monitor
	if len(monitors) > 0 {
		target = mon.bucketCount / len(monitors)
		if mon.bucketCount%len(monitors) != 0 {
			target++
		}
	}

	// load the current bucket allocations into the map
	for i, monitor := range doc.Monitor.Buckets {
		if buckets, found := m[monitor]; found && len(buckets) < target {
			// if the current bucket is allocated to a known monitor and doesn't
			// take its number of buckets above the target, keep it there...
			m[monitor] = append(m[monitor], i)
		} else {
			// ...otherwise we'll reallocate it below
			unallocated = append(unallocated, i)
		}
	}

	// reallocate all unallocated buckets, appending to the least loaded monitor
	if len(monitors) > 0 {
		for _, i := range unallocated {
			var leastMonitor string
			for monitor := range m {
				if leastMonitor == "" ||
					len(m[monitor]) < len(m[leastMonitor]) {
					leastMonitor = monitor
				}
			}

			m[leastMonitor] = append(m[leastMonitor], i)
		}
	}

	// write the updated bucket allocations back to the document
	for _, i := range unallocated {
		doc.Monitor.Buckets[i] = "" // should only happen if there are no known monitors
	}
	for monitor, buckets := range m {
		for _, i := range buckets {
			doc.Monitor.Buckets[i] = monitor
		}
	}
}
