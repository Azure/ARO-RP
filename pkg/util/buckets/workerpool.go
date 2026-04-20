package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"
)

// Workable is the interface that a database document must implement to be served by a WorkerPool.
type Workable interface {
	GetID() string
	GetKey() string
}

// WorkerFunc is the type for spawned workers. Workers are given a stop channel and the ID of the item to spawn.
type WorkerFunc func(<-chan struct{}, string)

type cacheDoc[E Workable] struct {
	doc  E
	stop chan<- struct{}
}

type workerPool[E Workable] struct {
	baseLog *logrus.Entry

	docs *xsync.Map[string, *cacheDoc[E]]

	spawnWorker WorkerFunc
	workerPool  sync.WaitGroup
}

type WorkerPool[E Workable] interface {
	StopAndWait()
	WaitForWorkerCompletion()
	CacheSize() int

	Doc(string) (E, bool)
	DeleteDoc(E)
	UpsertDoc(E)
}

func NewWorkerPool[W Workable](log *logrus.Entry, worker WorkerFunc) *workerPool[W] {
	return &workerPool[W]{
		baseLog: log,

		spawnWorker: worker,
		docs:        xsync.NewMap[string, *cacheDoc[W]](),
	}
}

// Return the size of the document cache.
func (mon *workerPool[E]) CacheSize() int {
	return mon.docs.Size()
}

func (mon *workerPool[E]) Doc(key string) (r E, ok bool) {
	key = strings.ToLower(key)

	v, ok := mon.docs.Load(key)
	if v == nil || !ok {
		mon.baseLog.Println("wanted ", key)
		for k, v := range mon.docs.All() {
			mon.baseLog.Println(k, v)
		}
		ok = false
		return
	}
	return v.doc, true
}

// DeleteDoc deletes the given document from c.docs, signalling the associated
// worker goroutine to stop if it exists.
func (c *workerPool[E]) DeleteDoc(doc E) {
	c.docs.Compute(doc.GetKey(), func(oldValue *cacheDoc[E], loaded bool) (newValue *cacheDoc[E], op xsync.ComputeOp) {
		if loaded && oldValue.stop != nil {
			close(oldValue.stop)
		}
		return nil, xsync.DeleteOp
	})
}

// UpsertDoc inserts or updates the given document into c.docs, calling fixDoc
// to potentially spawn a new worker.
func (c *workerPool[E]) UpsertDoc(doc E) {
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

// workerPool.fixDoc ensures that there is a worker goroutine for the given document.
func (c *workerPool[E]) fixDoc(v *cacheDoc[E]) {
	if v.stop == nil {
		c.baseLog.Debugf("spawning worker for %s", v.doc.GetKey())
		ch := make(chan struct{})
		v.stop = ch
		c.workerPool.Go(func() { c.spawnWorker(ch, v.doc.GetKey()) })
	}
}

// Stop stops all workers.
func (c *workerPool[E]) StopAndWait() {
	for _, v := range c.docs.All() {
		if v.stop != nil {
			close(v.stop)
			v.stop = nil
		}
	}
	c.workerPool.Wait()
}

func (c *workerPool[E]) WaitForWorkerCompletion() {
	c.workerPool.Wait()
}
