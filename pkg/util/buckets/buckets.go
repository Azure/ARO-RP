package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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

type monitor[E Bucketable] struct {
	baseLog *logrus.Entry

	bucketCount int
	buckets     map[int]struct{}

	mu   *sync.RWMutex
	docs map[string]*cacheDoc[E]

	worker WorkerFunc
}

type BucketWorker[E Bucketable] interface {
	Stop()
	SetBuckets([]int)

	Doc(string) (E, bool)
	DeleteDoc(E)
	UpsertDoc(E)
}

func NewBucketWorker[E Bucketable](log *logrus.Entry, worker WorkerFunc, mu *sync.RWMutex) *monitor[E] {
	return &monitor[E]{
		baseLog: log,

		worker: worker,
		docs:   map[string]*cacheDoc[E]{},

		buckets:     map[int]struct{}{},
		bucketCount: bucket.Buckets,

		mu: mu,
	}
}

func (mon *monitor[E]) Doc(id string) (r E, ok bool) {
	id = strings.ToLower(id)
	v := mon.docs[id]
	if v == nil {
		ok = false
		return
	}
	return v.doc, true
}

func (mon *monitor[E]) SetBuckets(buckets []int) {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	mon.buckets = map[int]struct{}{}

	for _, i := range buckets {
		mon.buckets[i] = struct{}{}
	}

	for _, v := range mon.docs {
		mon.FixDoc(v.doc)
	}
}
