package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"
)

// Bucketable is the interface that a database document must implement to be served by a BucketWorkerPool.
type Bucketable interface {
	Workable
	GetBucket() int
}

type bucketWorkerPool[E Bucketable] struct {
	workerPool[E]
	buckets map[int]struct{}

	bucketMu *sync.RWMutex
}

type BucketWorkerPool[E Bucketable] interface {
	WorkerPool[E]
	SetBuckets([]int)
	GetBuckets() []int
}

func NewBucketWorkerPool[E Bucketable](log *logrus.Entry, worker WorkerFunc) *bucketWorkerPool[E] {
	return &bucketWorkerPool[E]{
		workerPool: workerPool[E]{
			baseLog: log,

			spawnWorker: worker,
			docs:        xsync.NewMap[string, *cacheDoc[E]](),
			stopping:    &atomic.Bool{},
		},

		buckets:  map[int]struct{}{},
		bucketMu: &sync.RWMutex{},
	}
}

// Update the buckets that we want to pay attention to.
func (c *bucketWorkerPool[E]) SetBuckets(buckets []int) {
	c.bucketMu.Lock()
	defer c.bucketMu.Unlock()
	oldBuckets := c.buckets
	c.buckets = make(map[int]struct{}, len(buckets))

	for _, i := range buckets {
		c.buckets[i] = struct{}{}
	}

	if !reflect.DeepEqual(c.buckets, oldBuckets) {
		c.baseLog.Printf("servicing %d buckets", len(c.buckets))
		for _, v := range c.docs.All() {
			c.fixDoc(v)
		}
	}
}

// Return the buckets that are the responsibility of this responder.
func (c *bucketWorkerPool[E]) GetBuckets() []int {
	c.bucketMu.RLock()
	defer c.bucketMu.RUnlock()
	return slices.Collect(maps.Keys(c.buckets))
}

// BucketWorkerPool's UpsertDoc ensures that we have the bucket reader lock, so
// that fixDoc does not happen in the middle of a bucket update. This is
// duplicated from workerPool.UpsertDoc() as golang has no super()-equiv to
// allow that version of the func to call bucketWorkerPool's fixDoc
// preferentially.
func (c *bucketWorkerPool[E]) UpsertDoc(doc E) {
	c.bucketMu.RLock()
	defer c.bucketMu.RUnlock()
	c.docs.Compute(doc.GetKey(), func(oldValue *cacheDoc[E], loaded bool) (newValue *cacheDoc[E], op xsync.ComputeOp) {
		if loaded {
			newValue = &cacheDoc[E]{doc: doc, stop: oldValue.stop}
			c.fixDoc(newValue)
			return newValue, xsync.UpdateOp
		} else {
			newValue = &cacheDoc[E]{doc: doc}
			c.fixDoc(newValue)
			return newValue, xsync.UpdateOp
		}
	})
}

// bucketWorkerPool.fixDoc ensures that there is a worker goroutine for the
// given document if it is in a bucket owned by us. Caller needs to own
// c.bucketMu.
func (c *bucketWorkerPool[E]) fixDoc(v *cacheDoc[E]) {
	_, ours := c.buckets[v.doc.GetBucket()]

	if !ours && v.stop != nil {
		c.baseLog.Debugf("we no longer own cluster, closing worker for %s", v.doc.GetID())
		close(v.stop)
		v.stop = nil
	} else if ours {
		c.workerPool.fixDoc(v)
	}
}
