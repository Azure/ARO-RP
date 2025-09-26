package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type cacheDoc struct {
	doc  *api.OpenShiftClusterDocument
	stop chan<- struct{}
}

// deleteDoc deletes the given document from mon.docs, signalling the associated
// monitoring goroutine to stop if it exists.  Caller must hold mon.mu.Lock.
func (mon *monitor) deleteDoc(doc *api.OpenShiftClusterDocument) {
	v := mon.docs[doc.ID]

	if v != nil {
		if v.stop != nil {
			close(mon.docs[doc.ID].stop)
		}

		delete(mon.docs, doc.ID)
	}
}

// upsertDoc inserts or updates the given document into mon.docs, starting an
// associated monitoring goroutine if the document is in a bucket owned by us.
// Caller must hold mon.mu.Lock.
func (mon *monitor) upsertDoc(doc *api.OpenShiftClusterDocument) {
	v := mon.docs[doc.ID]

	if v == nil {
		v = &cacheDoc{}
		mon.docs[doc.ID] = v
	}

	v.doc = doc
	mon.fixDoc(doc)
}

// fixDocs ensures that there is a monitoring goroutine for all documents in all
// buckets owned by us.  Caller must hold mon.mu.Lock.
func (mon *monitor) fixDocs() {
	for _, v := range mon.docs {
		mon.fixDoc(v.doc)
	}
}

// fixDoc ensures that there is a monitoring goroutine for the given document
// iff it is in a bucket owned by us.  Caller must hold mon.mu.Lock.
func (mon *monitor) fixDoc(doc *api.OpenShiftClusterDocument) {
	v := mon.docs[doc.ID]
	_, ours := mon.buckets[v.doc.Bucket]

	if !ours && v.stop != nil {
		close(v.stop)
		v.stop = nil
	} else if ours && v.stop == nil {
		ch := make(chan struct{})
		v.stop = ch
		go mon.worker(ch, doc.ID)
	}
}
