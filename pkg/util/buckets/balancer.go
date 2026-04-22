package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

var (
	maxWorkerTTL      = 60 * time.Second
	maxWorkerInterval = 45 * time.Second
)

// Set maximums for workerTTL and Interval
func capIntervals(log *logrus.Entry, _interval time.Duration, _workerTTL time.Duration) (time.Duration, time.Duration) {
	workerTTL := _workerTTL
	interval := _interval

	if _interval > maxWorkerInterval {
		log.Errorf("interval must be at most %s to align with renewLease, was %s, capping", maxWorkerInterval, _interval)
		interval = maxWorkerInterval
	}
	if _workerTTL > maxWorkerTTL {
		log.Errorf("workerTTL must be at most %s to align with renewLease, was %s, capping", maxWorkerTTL, _workerTTL)
		workerTTL = maxWorkerTTL
	}
	if interval.Seconds() > workerTTL.Seconds()*0.75 {
		log.Errorf("interval %s was more than 75%% of TTL %s, capping", interval, workerTTL)
		interval = time.Duration(float64(workerTTL.Milliseconds())*0.75) * time.Millisecond
	}

	return interval, workerTTL
}

// Runs the bucket refresh loop. For a version that can be spawned in a
// goroutine directly, see StartBucketRefreshLoop.
func BucketRefreshLoop(
	ctx context.Context,
	log *logrus.Entry,
	workerType api.PoolWorkerType,
	bucketCount int,
	_interval time.Duration,
	_workerTTL time.Duration,
	dbPoolWorkers database.PoolWorkers,
	onBucketChange func([]int),
	stop <-chan struct{},
) error {
	interval, workerTTL := capIntervals(log, _interval, _workerTTL)

	t := time.NewTicker(interval)
	defer t.Stop()

	// We always need a master document to exist so that we can attempt to
	// dequeue it. If it already exists we will get a StatusPreconditionFailed
	// error, which is expected and we can ignore. The leasing of the master
	// document is in `tryMaster()`.
	_, err := dbPoolWorkers.Create(ctx, workerType, &api.PoolWorkerDocument{
		ID:         string(workerType),
		WorkerType: workerType,
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
		log.Error(fmt.Errorf("error bootstrapping master PoolWorkerDocument (not a 412): %w", err))
		return err
	}

	isMaster := false
	for {
		// register ourself as a worker, we need to refresh within workerTTL seconds
		err := dbPoolWorkers.PoolWorkerHeartbeat(ctx, workerType, int(workerTTL.Seconds()))
		if err != nil {
			log.Error(fmt.Errorf("error registering ourselves as a %s poolWorker, continuing: %w", workerType, err))
		}

		isMaster, err = tryMaster(ctx, log, workerType, bucketCount, dbPoolWorkers, isMaster)
		if err != nil {
			log.Error(fmt.Errorf("error registering ourselves as the master, continuing: %w", err))
		}

		buckets, err := dbPoolWorkers.ListBuckets(ctx, workerType)
		if err != nil {
			log.Error(fmt.Errorf("error reading bucket allocation from master: %w", err))
		} else {
			onBucketChange(buckets)
		}

		if err = ctx.Err(); err != nil {
			return err
		}

		select {
		case <-t.C:
		case <-stop:
			return nil
		}
	}
}

// master updates the PoolWorkerDocument with the list of buckets balanced between
// registered workers
func tryMaster(
	ctx context.Context,
	log *logrus.Entry,
	workerType api.PoolWorkerType,
	bucketCount int,
	dbPoolWorkers database.PoolWorkers,
	isMaster bool,
) (bool, error) {
	// if we know we're not the master, attempt to gain the lease on the
	// PoolWorkerDocument
	if !isMaster {
		doc, err := dbPoolWorkers.TryLease(ctx, workerType)
		if err == nil && doc == nil {
			// We didn't become the master
			return false, nil
		} else if err != nil {
			log.Errorf("problem when trying lease: err: %s", err.Error())
			return false, err
		}
		isMaster = true
		log.Infof("became the %s master", workerType)
	}

	// we know we're not the master; give up
	if !isMaster {
		return false, nil
	}

	// we think we're the master.  Gather up all the registered workers
	// including ourself, balance buckets between them and write the bucket
	// allocations to the database.  If it turns out that we're not the master,
	// the patch will fail
	_, err := dbPoolWorkers.PatchWithLease(ctx, workerType, string(workerType), func(doc *api.PoolWorkerDocument) error {
		docs, err := dbPoolWorkers.ListPoolWorkers(ctx, workerType)
		if err != nil {
			return err
		}

		var workers []string
		if docs != nil {
			workers = make([]string, 0, len(docs.PoolWorkerDocuments))
			for _, doc := range docs.PoolWorkerDocuments {
				workers = append(workers, doc.ID)
			}
		}

		log.Debugf("workers: %v", workers)

		balance(workers, bucketCount, doc)
		return nil
	})
	if err != nil && err.Error() == "lost lease" {
		isMaster = false
		log.Infof("stopped being the %s master", workerType)
	}
	return isMaster, err
}

// balance shares out buckets over a slice of registered workers
func balance(workers []string, bucketCount int, doc *api.PoolWorkerDocument) {
	// initialise doc.PoolWorker
	if doc.PoolWorker == nil {
		doc.PoolWorker = &api.PoolWorker{}
	}

	// ensure len(doc.PoolWorker.Buckets) == mon.bucketCount: this should only do
	// anything on the very first run
	if len(doc.PoolWorker.Buckets) < bucketCount {
		doc.PoolWorker.Buckets = append(doc.PoolWorker.Buckets, make([]string, bucketCount-len(doc.PoolWorker.Buckets))...)
	}
	if len(doc.PoolWorker.Buckets) > bucketCount { // should never happen
		doc.PoolWorker.Buckets = doc.PoolWorker.Buckets[:bucketCount]
	}

	var unallocated []int
	m := make(map[string][]int, len(workers)) // map of worker to list of buckets it owns
	for _, worker := range workers {
		m[worker] = nil
	}

	var target int // target number of buckets per worker
	if len(workers) > 0 {
		target = bucketCount / len(workers)
		if bucketCount%len(workers) != 0 {
			target++
		}
	}

	// load the current bucket allocations into the map
	for i, worker := range doc.PoolWorker.Buckets {
		if buckets, found := m[worker]; found && len(buckets) < target {
			// if the current bucket is allocated to a known worker and doesn't
			// take its number of buckets above the target, keep it there...
			m[worker] = append(m[worker], i)
		} else {
			// ...otherwise we'll reallocate it below
			unallocated = append(unallocated, i)
		}
	}

	// reallocate all unallocated buckets, appending to the least loaded worker
	if len(workers) > 0 {
		for _, i := range unallocated {
			var leastWorker string
			for worker := range m {
				if leastWorker == "" ||
					len(m[worker]) < len(m[leastWorker]) {
					leastWorker = worker
				}
			}

			m[leastWorker] = append(m[leastWorker], i)
		}
	}

	// write the updated bucket allocations back to the document
	for _, i := range unallocated {
		doc.PoolWorker.Buckets[i] = "" // should only happen if there are no known workers
	}
	for worker, buckets := range m {
		for _, i := range buckets {
			doc.PoolWorker.Buckets[i] = worker
		}
	}
}

// Start the bucket refreshing loop, logging and calling onError() on startup
// failure. bucketUpdateStartedOrErrored will complete when the worker has
// failed, or the initial master allocation has succeeded.
func StartBucketRefreshLoop(
	ctx context.Context,
	log *logrus.Entry,
	workerType api.PoolWorkerType,
	bucketCount int,
	refreshInterval time.Duration,
	workerDocTTL time.Duration,
	db database.PoolWorkers,
	onBucketChange func([]int),
	stop <-chan struct{},
	onError context.CancelCauseFunc,
	bucketUpdateStartedOrErrored *sync.WaitGroup,
) {
	defer recover.Panic(log)

	// We don't use wg.Go() here as we want Done() to happen when either it
	// starts or the whole process fails
	_signalWg := sync.OnceFunc(bucketUpdateStartedOrErrored.Done)

	// If we error out/finish, signal the waitgroup
	defer _signalWg()

	_onBucketChange := func(i []int) {
		onBucketChange(i)
		_signalWg()
	}

	_err := BucketRefreshLoop(ctx, log, workerType, bucketCount, refreshInterval, workerDocTTL, db, _onBucketChange, stop)
	if _err != nil {
		log.Errorf("unable to start bucket worker, exiting: %s", _err.Error())
		onError(_err)
	}
}
