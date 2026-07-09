package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
)

// WorkerFunc is the type for spawned workers. Workers are given a stop channel
// and the ID of the item to spawn.
type WorkerFunc func(<-chan struct{}, string)

type cacheDoc[E database.Keyable] struct {
	doc  E
	stop chan<- struct{}
}

type workerPool[E database.Keyable] struct {
	baseLog *logrus.Entry

	canonicalize func(string) string
	docs         *xsync.Map[string, *cacheDoc[E]]

	spawnWorker WorkerFunc
	workerPool  sync.WaitGroup
	stopping    *atomic.Bool
}

type WorkerPool[E database.Keyable] interface {
	StopAndWait()
	WaitForWorkerCompletion()
	CacheSize() int

	Doc(string) (E, bool)
	DeleteDoc(E)
	UpsertDoc(E)
}

func NewWorkerPool[W database.Keyable](log *logrus.Entry, worker WorkerFunc) *workerPool[W] {
	return &workerPool[W]{
		baseLog: log,

		canonicalize: strings.ToLower,
		docs:         xsync.NewMap[string, *cacheDoc[W]](),

		spawnWorker: worker,
		stopping:    &atomic.Bool{},
	}
}

// Return the size of the document cache.
func (c *workerPool[E]) CacheSize() int {
	return c.docs.Size()
}

func (c *workerPool[E]) Doc(key string) (r E, ok bool) {
	v, ok := c.docs.Load(c.canonicalize(key))
	if v == nil || !ok {
		ok = false
		return
	}
	return v.doc, true
}

// DeleteDoc deletes the given document from c.docs, signalling the associated
// worker goroutine to stop if it exists.
func (c *workerPool[E]) DeleteDoc(doc E) {
	c.docs.Compute(c.canonicalize(doc.GetKey()), func(oldValue *cacheDoc[E], loaded bool) (newValue *cacheDoc[E], op xsync.ComputeOp) {
		if loaded && oldValue.stop != nil {
			close(oldValue.stop)
		}
		return nil, xsync.DeleteOp
	})
}

// UpsertDoc inserts or updates the given document into c.docs, calling fixDoc
// to potentially spawn a new worker.
func (c *workerPool[E]) UpsertDoc(doc E) {
	c.docs.Compute(c.canonicalize(doc.GetKey()), func(oldValue *cacheDoc[E], loaded bool) (newValue *cacheDoc[E], op xsync.ComputeOp) {
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

// workerPool.fixDoc ensures that there is a worker goroutine for the given document.
func (c *workerPool[E]) fixDoc(v *cacheDoc[E]) {
	if v.stop == nil && !c.stopping.Load() {
		key := c.canonicalize(v.doc.GetKey())
		c.baseLog.Debugf("spawning worker for %s", key)
		ch := make(chan struct{})
		v.stop = ch
		c.workerPool.Go(func() { c.spawnWorker(ch, key) })
	}
}

// Stop stops all workers.
func (c *workerPool[E]) StopAndWait() {
	c.stopping.Store(true)
	for _, v := range c.docs.All() {
		c.stopWorker(v.doc)
	}
	c.workerPool.Wait()
}

func (c *workerPool[E]) WaitForWorkerCompletion() {
	c.workerPool.Wait()
}

func (c *workerPool[E]) stopWorker(doc E) {
	// Use .Compute() which uses the internal lock kept by xsync.Map rather than
	// having a second locking arrangement
	c.docs.Compute(c.canonicalize(doc.GetKey()), func(oldValue *cacheDoc[E], loaded bool) (newValue *cacheDoc[E], op xsync.ComputeOp) {
		if loaded && oldValue.stop != nil {
			close(oldValue.stop)
			oldValue.stop = nil
		}
		return nil, xsync.CancelOp
	})
}
