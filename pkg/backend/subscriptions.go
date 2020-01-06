package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type subscriptionBackend struct {
	*backend
}

// try tries to dequeue an SubscriptionDocument for work, and works it on a new
// goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (sb *subscriptionBackend) try() (bool, error) {
	doc, err := sb.db.Subscriptions.Dequeue()
	if err != nil || doc == nil {
		return false, err
	}

	log := sb.baseLog.WithField("subscription", doc.ID)
	if doc.Dequeues > maxDequeueCount {
		log.Errorf("dequeued %d times, failing", doc.Dequeues)
		return true, sb.endLease(nil, doc, false, true)
	}

	log.Print("dequeued")
	atomic.AddInt32(&sb.workers, 1)
	go func() {
		defer recover.Panic(log)

		defer func() {
			atomic.AddInt32(&sb.workers, -1)
			sb.cond.Signal()
		}()

		t := time.Now()

		err := sb.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}

		log.WithField("durationMs", int(time.Now().Sub(t)/time.Millisecond)).Print("done")
	}()

	return true, nil
}

// handle is responsible for handling backend operation and lease
func (sb *subscriptionBackend) handle(ctx context.Context, log *logrus.Entry, doc *api.SubscriptionDocument) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := sb.heartbeat(cancel, log, doc)
	defer stop()

	done, err := sb.handleDelete(ctx, log, doc)
	if err != nil {
		log.Error(err)
		return sb.endLease(stop, doc, false, false)
	}

	return sb.endLease(stop, doc, done, !done)
}

// handleDelete ensures that all the clusters in a subscription which is being
// deleted are at least enqueued for deletion.  It returns a boolean to the
// caller indicating whether it this is the case - if this is false, the caller
// should sleep before calling again
func (sb *subscriptionBackend) handleDelete(ctx context.Context, log *logrus.Entry, subdoc *api.SubscriptionDocument) (bool, error) {
	i, err := sb.db.OpenShiftClusters.ListByPrefix(subdoc.ID, "/subscriptions/"+subdoc.ID+"/")
	if err != nil {
		return false, err
	}

	done := true
	for {
		docs, err := i.Next()
		if err != nil {
			return false, err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			_, err = sb.db.OpenShiftClusters.Patch(doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				switch doc.OpenShiftCluster.Properties.ProvisioningState {
				case api.ProvisioningStateCreating,
					api.ProvisioningStateUpdating:
					done = false
				case api.ProvisioningStateDeleting:
					// nothing to do
				case api.ProvisioningStateSucceeded,
					api.ProvisioningStateFailed:
					doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateDeleting
				default:
					return fmt.Errorf("unexpected provisioningState %q", doc.OpenShiftCluster.Properties.ProvisioningState)
				}
				return nil
			})
			if err != nil {
				return false, err
			}
		}
	}

	return done, nil
}

func (sb *subscriptionBackend) heartbeat(cancel context.CancelFunc, log *logrus.Entry, doc *api.SubscriptionDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := sb.db.Subscriptions.Lease(doc.ID)
			if err != nil {
				log.Error(err)
				cancel()
				return
			}

			select {
			case <-t.C:
			case <-stop:
				return
			}
		}
	}()

	return func() {
		if !stopped {
			close(stop)
			<-done
			stopped = true
		}
	}
}

func (sb *subscriptionBackend) endLease(stop func(), doc *api.SubscriptionDocument, done, retryLater bool) error {
	if stop != nil {
		stop()
	}

	_, err := sb.db.Subscriptions.EndLease(doc.ID, done, retryLater)
	return err
}
