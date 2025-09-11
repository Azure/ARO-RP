package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
)

type cacheDoc struct {
	doc  *api.OpenShiftClusterDocument
	stop chan<- struct{}
}

// deleteDoc deletes the given document from mon.docs, signalling the associated
// monitoring goroutine to stop if it exists.  Caller must hold mon.mu.Lock.
func (mon *monitor) DeleteDoc(doc *api.OpenShiftClusterDocument) {
	id := strings.ToLower(doc.ID)
	v := mon.docs[id]

	if v != nil {
		if v.stop != nil {
			close(mon.docs[id].stop)
		}

		delete(mon.docs, id)
	}
}

// upsertDoc inserts or updates the given document into mon.docs, starting an
// associated monitoring goroutine if the document is in a bucket owned by us.
// Caller must hold mon.mu.Lock.
func (mon *monitor) UpsertDoc(doc *api.OpenShiftClusterDocument) {
	id := strings.ToLower(doc.ID)
	v := mon.docs[id]

	if v == nil {
		v = &cacheDoc{}
		mon.docs[id] = v
	}

	v.doc = doc
	mon.FixDoc(doc)
}

// fixDoc ensures that there is a monitoring goroutine for the given document
// iff it is in a bucket owned by us.  Caller must hold mon.mu.Lock.
func (mon *monitor) FixDoc(doc *api.OpenShiftClusterDocument) {
	id := strings.ToLower(doc.ID)
	v := mon.docs[id]

	mon.baseLog.Debugf("fixing doc %s (%s)", doc.ID, doc.Key)
	_, ours := mon.buckets[v.doc.Bucket]

	if !ours && v.stop != nil {
		mon.baseLog.Debugf("stopping channel for %s", doc.ID)
		close(v.stop)
		v.stop = nil
	} else if ours && v.stop == nil {
		ch := make(chan struct{})
		v.stop = ch

		mon.baseLog.Debugf("spawning worker for %s", doc.ID)
		mon.worker(ch, doc.Key)
	}
}

// Stop stops all workers.
func (mon *monitor) Stop() {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	for _, v := range mon.docs {
		if v.stop != nil {
			close(v.stop)
			v.stop = nil
		}
	}
}
