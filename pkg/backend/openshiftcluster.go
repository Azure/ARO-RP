package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/backend/openshiftcluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type openShiftClusterBackend struct {
	*backend
}

// try tries to dequeue an OpenShiftClusterDocument for work, and works it on a
// new goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (ocb *openShiftClusterBackend) try(ctx context.Context) (bool, error) {
	doc, err := ocb.db.OpenShiftClusters.Dequeue(ctx)
	if err != nil || doc == nil {
		return false, err
	}

	log := ocb.baseLog
	log = utillog.EnrichWithResourceID(log, doc.OpenShiftCluster.ID)
	log = utillog.EnrichWithCorrelationData(log, doc.CorrelationData)

	if doc.Dequeues > maxDequeueCount {
		err := fmt.Errorf("dequeued %d times, failing", doc.Dequeues)
		return true, ocb.endLease(ctx, log, nil, doc, api.ProvisioningStateFailed, err)
	}

	log.Print("dequeued")
	atomic.AddInt32(&ocb.workers, 1)
	ocb.m.EmitGauge("backend.openshiftcluster.workers.count", int64(atomic.LoadInt32(&ocb.workers)), nil)

	go func() {
		defer recover.Panic(log)

		t := time.Now()

		defer func() {
			atomic.AddInt32(&ocb.workers, -1)
			ocb.m.EmitGauge("backend.openshiftcluster.workers.count", int64(atomic.LoadInt32(&ocb.workers)), nil)
			ocb.cond.Signal()

			ocb.m.EmitGauge("backend.openshiftcluster.duration", time.Now().Sub(t).Milliseconds(), map[string]string{
				"state": string(doc.OpenShiftCluster.Properties.ProvisioningState),
			})

			ocb.m.EmitGauge("backend.openshiftcluster.count", 1, map[string]string{
				"state": string(doc.OpenShiftCluster.Properties.ProvisioningState),
			})

			log.WithField("duration", time.Now().Sub(t).Seconds()).Print("done")
		}()

		err := ocb.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}

	}()

	return true, nil
}

// handle is responsible for handling backend operation and lease
func (ocb *openShiftClusterBackend) handle(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := ocb.heartbeat(ctx, cancel, log, doc)
	defer stop()

	m, err := openshiftcluster.NewManager(log, ocb.env, ocb.db.OpenShiftClusters, ocb.db.Billing, ocb.db.Subscriptions, doc)
	if err != nil {
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateCreating:
		log.Print("creating")

		err = m.Create(ctx)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		// re-get document and check the state:
		// if Install = nil, we are done with the install.
		// if Install != nil, we need to terminate, release lease and let other
		// backend worker to pick up next install phase
		doc, err = ocb.db.OpenShiftClusters.Get(ctx, strings.ToLower(doc.OpenShiftCluster.ID))
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		if doc.OpenShiftCluster.Properties.Install == nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateSucceeded, nil)
		}
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateCreating, nil)

	case api.ProvisioningStateUpdating, api.ProvisioningStateAdminUpdating:
		log.Print("updating")

		err = m.Update(ctx)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateSucceeded, nil)

	case api.ProvisioningStateDeleting:
		log.Print("deleting")

		err = m.Delete(ctx)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}

		err = ocb.updateAsyncOperation(ctx, log, doc.AsyncOperationID, nil, api.ProvisioningStateSucceeded, "", nil)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}

		stop()

		return ocb.db.OpenShiftClusters.Delete(ctx, doc)
	}

	return fmt.Errorf("unexpected provisioningState %q", doc.OpenShiftCluster.Properties.ProvisioningState)
}

func (ocb *openShiftClusterBackend) heartbeat(ctx context.Context, cancel context.CancelFunc, log *logrus.Entry, doc *api.OpenShiftClusterDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := ocb.db.OpenShiftClusters.Lease(ctx, doc.Key)
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

func (ocb *openShiftClusterBackend) updateAsyncOperation(ctx context.Context, log *logrus.Entry, id string, oc *api.OpenShiftCluster, provisioningState, failedProvisioningState api.ProvisioningState, backendErr error) error {
	if id != "" {
		_, err := ocb.db.AsyncOperations.Patch(ctx, id, func(asyncdoc *api.AsyncOperationDocument) error {
			asyncdoc.AsyncOperation.ProvisioningState = provisioningState

			now := time.Now()
			asyncdoc.AsyncOperation.EndTime = &now

			if provisioningState == api.ProvisioningStateFailed {
				// if type is CloudError - we want to propagate it to the
				// asyncOperations errors. Otherwise - return generic error
				err, ok := backendErr.(*api.CloudError)
				if ok {
					log.Print(backendErr)
					asyncdoc.AsyncOperation.Error = err.CloudErrorBody
				} else {
					log.Error(backendErr)
					asyncdoc.AsyncOperation.Error = &api.CloudErrorBody{
						Code:    api.CloudErrorCodeInternalServerError,
						Message: "Internal server error.",
					}
				}
			}

			if oc != nil {
				ocCopy := *oc
				ocCopy.Properties.ProvisioningState = provisioningState
				ocCopy.Properties.FailedProvisioningState = failedProvisioningState

				asyncdoc.OpenShiftCluster = &ocCopy
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (ocb *openShiftClusterBackend) endLease(ctx context.Context, log *logrus.Entry, stop func(), doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState, backendErr error) error {
	var failedProvisioningState api.ProvisioningState
	if provisioningState == api.ProvisioningStateFailed {
		failedProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	}

	// If cluster is in the non-terminal state we are still in the same
	// operational context and AsyncOperation should not be updated.
	if provisioningState.IsTerminal() {
		err := ocb.updateAsyncOperation(ctx, log, doc.AsyncOperationID, doc.OpenShiftCluster, provisioningState, failedProvisioningState, backendErr)
		if err != nil {
			return err
		}
	}

	if stop != nil {
		stop()
	}

	_, err := ocb.db.OpenShiftClusters.EndLease(ctx, doc.Key, provisioningState, failedProvisioningState)
	return err
}
