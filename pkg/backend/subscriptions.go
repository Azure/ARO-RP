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

func newSubscriptionBackend(b *backend) *subscriptionBackend {
	return &subscriptionBackend{backend: b}
}

// try tries to dequeue an SubscriptionDocument for work, and works it on a new
// goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (sb *subscriptionBackend) try(ctx context.Context) (bool, error) {
	doc, err := sb.dbSubscriptions.Dequeue(ctx)
	if err != nil || doc == nil {
		return false, err
	}

	log := sb.baseLog.WithField("subscription", doc.ID)
	if doc.Dequeues > maxDequeueCount {
		log.Errorf("dequeued %d times, failing", doc.Dequeues)
		return true, sb.endLease(ctx, nil, doc, false, true)
	}

	log.Print("dequeued")
	atomic.AddInt32(&sb.workers, 1)
	sb.m.EmitGauge("backend.subscriptions.workers.count", int64(atomic.LoadInt32(&sb.workers)), nil)

	go func() {
		defer recover.Panic(log)

		t := time.Now()

		defer func() {
			atomic.AddInt32(&sb.workers, -1)
			sb.m.EmitGauge("backend.subscriptions.workers.count", int64(atomic.LoadInt32(&sb.workers)), nil)
			sb.cond.Signal()

			sb.m.EmitGauge("backend.subscriptions.duration", time.Since(t).Milliseconds(), map[string]string{
				"state": string(doc.Subscription.State),
			})

			sb.m.EmitGauge("backend.subscriptions.count", 1, nil)

			log.WithField("duration", time.Since(t).Seconds()).Print("done")
		}()

		err := sb.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}
	}()

	return true, nil
}

// handle is responsible for handling backend operation and lease
func (sb *subscriptionBackend) handle(ctx context.Context, log *logrus.Entry, doc *api.SubscriptionDocument) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := sb.heartbeat(ctx, cancel, log, doc)
	defer stop()

	done, err := sb.handleDelete(ctx, log, doc)
	if err != nil {
		log.Error(err)
		return sb.endLease(ctx, stop, doc, false, false)
	}

	return sb.endLease(ctx, stop, doc, done, !done)
}

// handleDelete ensures that all the clusters in a subscription which is being
// deleted are at least enqueued for deletion.  It returns a boolean to the
// caller indicating whether it this is the case - if this is false, the caller
// should sleep before calling again
func (sb *subscriptionBackend) handleDelete(ctx context.Context, log *logrus.Entry, subdoc *api.SubscriptionDocument) (bool, error) {
	// at the time of writing, subscription docs are only enqueued to enable
	// cascading delete, but for safety let's double-check our state here before
	// actually deleting anything in case the above assumption ever changes...
	if subdoc.Subscription.State != api.SubscriptionStateDeleted {
		return false, fmt.Errorf("handleDelete was called, but subscription is in state %s", subdoc.Subscription.State)
	}

	i, err := sb.dbOpenShiftClusters.ListByPrefix(subdoc.ID, "/subscriptions/"+subdoc.ID+"/", "")
	if err != nil {
		return false, err
	}

	done := true
	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return false, err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			_, err = sb.dbOpenShiftClusters.Patch(ctx, doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				switch doc.OpenShiftCluster.Properties.ProvisioningState {
				case api.ProvisioningStateCreating,
					api.ProvisioningStateUpdating,
					api.ProvisioningStateAdminUpdating:
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

func (sb *subscriptionBackend) heartbeat(ctx context.Context, cancel context.CancelFunc, log *logrus.Entry, doc *api.SubscriptionDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := sb.dbSubscriptions.Lease(ctx, doc.ID)
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

func (sb *subscriptionBackend) endLease(ctx context.Context, stop func(), doc *api.SubscriptionDocument, done, retryLater bool) error {
	if stop != nil {
		stop()
	}

	_, err := sb.dbSubscriptions.EndLease(ctx, doc.ID, done, retryLater)
	return err
}
