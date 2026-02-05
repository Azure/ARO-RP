package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

type cacheDoc[E IDer] struct {
	doc  E
	stop chan<- struct{}
}

// deleteDoc deletes the given document from mon.docs, signalling the associated
// monitoring goroutine to stop if it exists.  Caller must hold mon.mu.Lock.
func (mon *monitor[E]) DeleteDoc(doc E) {
	id := strings.ToLower(doc.GetID())
	v := mon.docs[id]

	if v != nil {
		if v.stop != nil {
			mon.baseLog.Debugf("deleting doc, closing worker for %s", doc.GetID())
			close(mon.docs[id].stop)
		}

		delete(mon.docs, id)
	}
}

// upsertDoc inserts or updates the given document into mon.docs, starting an
// associated monitoring goroutine if the document is in a bucket owned by us.
// Caller must hold mon.mu.Lock.
func (mon *monitor[E]) UpsertDoc(doc E) {
	id := strings.ToLower(doc.GetID())
	v := mon.docs[id]

	if v == nil {
		v = &cacheDoc[E]{}
		mon.docs[id] = v
	}

	v.doc = doc
	mon.FixDoc(doc)
}

// fixDoc ensures that there is a monitoring goroutine for the given document
// iff it is in a bucket owned by us.  Caller must hold mon.mu.Lock.
func (mon *monitor[E]) FixDoc(doc E) {
	id := strings.ToLower(doc.GetID())
	v := mon.docs[id]

	_, ours := mon.buckets[v.doc.GetBucket()]

	if !ours && v.stop != nil {
		mon.baseLog.Debugf("we no longer own cluster, closing worker for %s", doc.GetID())
		close(v.stop)
		v.stop = nil
	} else if ours && v.stop == nil {
		ch := make(chan struct{})
		v.stop = ch

		mon.baseLog.Debugf("spawning worker for %s", doc.GetID())
		mon.worker(ch, doc.GetKey())
	}
}

// Stop stops all workers.
func (mon *monitor[E]) Stop() {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	for _, v := range mon.docs {
		if v.stop != nil {
			close(v.stop)
			v.stop = nil
		}
	}
}
