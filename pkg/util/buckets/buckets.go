package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/bucket"
)

type Bucketable interface {
	GetID() string
	GetBucket() int
	GetKey() string
}

type WorkerFunc func(<-chan struct{}, string)

type cache[E Bucketable] struct {
	baseLog *logrus.Entry

	bucketCount int
	buckets     map[int]struct{}

	mu   *sync.RWMutex
	docs map[string]*cacheDoc[E]

	worker WorkerFunc
}

type BucketWorker[E Bucketable] interface {
	Stop()
	Size() int
	SetBuckets([]int)

	Doc(string) (E, bool)
	DeleteDoc(E)
	UpsertDoc(E)
}

func NewBucketWorker[E Bucketable](log *logrus.Entry, worker WorkerFunc, mu *sync.RWMutex) *cache[E] {
	return &cache[E]{
		baseLog: log,

		worker: worker,
		docs:   map[string]*cacheDoc[E]{},

		buckets:     map[int]struct{}{},
		bucketCount: bucket.Buckets,

		mu: mu,
	}
}

// Return the size of the document cache. Caller must hold mon.mu.
func (mon *cache[E]) Size() int {
	return len(mon.docs)
}

func (mon *cache[E]) Doc(id string) (r E, ok bool) {
	mon.mu.RLock()
	defer mon.mu.RUnlock()
	id = strings.ToLower(id)
	v := mon.docs[id]
	if v == nil {
		ok = false
		return
	}
	return v.doc, true
}

func (mon *cache[E]) SetBuckets(buckets []int) {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	oldBuckets := mon.buckets
	mon.buckets = make(map[int]struct{}, len(buckets))

	for _, i := range buckets {
		mon.buckets[i] = struct{}{}
	}

	if !reflect.DeepEqual(mon.buckets, oldBuckets) {
		mon.baseLog.Printf("servicing %d buckets", len(mon.buckets))
		for _, v := range mon.docs {
			mon.FixDoc(v.doc)
		}
	}
}

type cacheDoc[E Bucketable] struct {
	doc  E
	stop chan<- struct{}
}

// deleteDoc deletes the given document from mon.docs, signalling the associated
// monitoring goroutine to stop if it exists.  Caller must hold mon.mu.Lock.
func (mon *cache[E]) DeleteDoc(doc E) {
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
func (mon *cache[E]) UpsertDoc(doc E) {
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
// if it is in a bucket owned by us.  Caller must hold mon.mu.Lock.
func (mon *cache[E]) FixDoc(doc E) {
	id := strings.ToLower(doc.GetID())
	v := mon.docs[id]

	var ours bool
	// getBucket() with -1 is served by all
	if v.doc.GetBucket() > -1 {
		_, ours = mon.buckets[v.doc.GetBucket()]
	} else {
		ours = true
	}

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
func (mon *cache[E]) Stop() {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	for _, v := range mon.docs {
		if v.stop != nil {
			close(v.stop)
			v.stop = nil
		}
	}
}
