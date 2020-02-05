package subscriptions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const maxDequeueCount = 5

type SubscriptionBackend struct {
	env env.Interface
	db  *database.Database
	m   metrics.Interface
}

func New(env env.Interface, db *database.Database, m metrics.Interface) *SubscriptionBackend {
	return &SubscriptionBackend{
		env: env,
		db:  db,
		m:   m,
	}
}

func (sb *SubscriptionBackend) Dequeue() func(context.Context) (interface{}, error) {
	return sb.db.Subscriptions.DequeueRaw
}

// Handle is responsible for handling backend operation all related operations
// to this backend
func (sb *SubscriptionBackend) Handle(ctx context.Context, baseLog *logrus.Entry, docRaw interface{}) error {
	t := time.Now()
	doc := docRaw.(*api.SubscriptionDocument)
	log := baseLog.WithField("subscription", doc.ID)

	defer func() {
		sb.m.EmitFloat("backend.subscriptions.duration", time.Now().Sub(t).Seconds(), map[string]string{
			"state": string(doc.Subscription.State),
		})

		sb.m.EmitGauge("backend.subscriptions.count", 1, map[string]string{
			"state": string(doc.Subscription.State),
		})
	}()

	if doc.Dequeues > maxDequeueCount {
		log.Errorf("dequeued %d times, failing", doc.Dequeues)
		return sb.endLease(ctx, nil, doc, false, true)
	}

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
func (sb *SubscriptionBackend) handleDelete(ctx context.Context, log *logrus.Entry, subdoc *api.SubscriptionDocument) (bool, error) {
	i, err := sb.db.OpenShiftClusters.ListByPrefix(subdoc.ID, "/subscriptions/"+subdoc.ID+"/")
	if err != nil {
		return false, err
	}

	done := true
	for {
		docs, err := i.Next(ctx)
		if err != nil {
			return false, err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			_, err = sb.db.OpenShiftClusters.Patch(ctx, doc.Key, func(doc *api.OpenShiftClusterDocument) error {
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

func (sb *SubscriptionBackend) heartbeat(ctx context.Context, cancel context.CancelFunc, log *logrus.Entry, doc *api.SubscriptionDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := sb.db.Subscriptions.Lease(ctx, doc.ID)
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

func (sb *SubscriptionBackend) endLease(ctx context.Context, stop func(), doc *api.SubscriptionDocument, done, retryLater bool) error {
	if stop != nil {
		stop()
	}

	_, err := sb.db.Subscriptions.EndLease(ctx, doc.ID, done, retryLater)
	return err
}
