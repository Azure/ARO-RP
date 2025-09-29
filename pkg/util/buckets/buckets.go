package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
)

type WorkerFunc func(<-chan struct{}, string)

type monitor struct {
	baseLog *logrus.Entry

	bucketCount int
	buckets     map[int]struct{}

	mu   *sync.RWMutex
	docs map[string]*cacheDoc

	worker WorkerFunc
}

type BucketWorker interface {
	Stop()
	SetBuckets([]int)

	Doc(string) *api.OpenShiftClusterDocument
	DeleteDoc(*api.OpenShiftClusterDocument)
	UpsertDoc(*api.OpenShiftClusterDocument)
}

func NewBucketWorker(log *logrus.Entry, worker WorkerFunc, mu *sync.RWMutex) *monitor {
	return &monitor{
		baseLog: log,

		worker: worker,
		docs:   map[string]*cacheDoc{},

		buckets:     map[int]struct{}{},
		bucketCount: bucket.Buckets,

		mu: mu,
	}
}

func (mon *monitor) Doc(id string) *api.OpenShiftClusterDocument {
	id = strings.ToLower(id)
	v := mon.docs[id]
	if v == nil {
		return nil
	}
	return v.doc
}

func (mon *monitor) SetBuckets(buckets []int) {
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
