package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type openShiftClusterBackend struct {
	*backend

	newManager func(context.Context, *logrus.Entry, env.Interface, database.OpenShiftClusters, database.Gateway, encryption.AEAD, billing.Manager, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) (cluster.Interface, error)
}

func newOpenShiftClusterBackend(b *backend) *openShiftClusterBackend {
	return &openShiftClusterBackend{
		backend:    b,
		newManager: cluster.New,
	}
}

// try tries to dequeue an OpenShiftClusterDocument for work, and works it on a
// new goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (ocb *openShiftClusterBackend) try(ctx context.Context) (bool, error) {
	doc, err := ocb.dbOpenShiftClusters.Dequeue(ctx)
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

			log.WithField("duration", time.Since(t).Seconds()).Print("done")
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

	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return err
	}

	subscriptionDoc, err := ocb.dbSubscriptions.Get(ctx, r.SubscriptionID)
	if err != nil {
		return err
	}

	m, err := ocb.newManager(ctx, log, ocb.env, ocb.dbOpenShiftClusters, ocb.dbGateway, ocb.aead, ocb.billing, doc, subscriptionDoc)
	if err != nil {
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateCreating:
		log.Print("creating")

		err = m.Install(ctx)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		// re-get document and check the state:
		// if Install = nil, we are done with the install.
		// if Install != nil, we need to terminate, release lease and let other
		// backend worker to pick up next install phase
		doc, err = ocb.dbOpenShiftClusters.Get(ctx, strings.ToLower(doc.OpenShiftCluster.ID))
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		if doc.OpenShiftCluster.Properties.Install == nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateSucceeded, nil)
		}
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateCreating, nil)

	case api.ProvisioningStateAdminUpdating:
		log.Printf("admin updating (type: %s)", doc.OpenShiftCluster.Properties.MaintenanceTask)

		err = m.AdminUpdate(ctx)
		if err != nil {
			return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateFailed, err)
		}
		return ocb.endLease(ctx, log, stop, doc, api.ProvisioningStateSucceeded, nil)

	case api.ProvisioningStateUpdating:
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

		return ocb.dbOpenShiftClusters.Delete(ctx, doc)
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
			_, err := ocb.dbOpenShiftClusters.Lease(ctx, doc.Key)
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
		_, err := ocb.dbAsyncOperations.Patch(ctx, id, func(asyncdoc *api.AsyncOperationDocument) error {
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
				ocCopy.Properties.LastProvisioningState = ""
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
	var adminUpdateError *string
	var failedProvisioningState api.ProvisioningState

	if doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateAdminUpdating &&
		provisioningState == api.ProvisioningStateFailed {
		failedProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	}

	// If cluster is in the non-terminal state we are still in the same
	// operational context and AsyncOperation should not be updated.
	if provisioningState.IsTerminal() {
		err := ocb.updateAsyncOperation(ctx, log, doc.AsyncOperationID, doc.OpenShiftCluster, provisioningState, failedProvisioningState, backendErr)
		if err != nil {
			return err
		}

		ocb.emitMetrics(doc, provisioningState)
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateAdminUpdating {
		provisioningState = doc.OpenShiftCluster.Properties.LastProvisioningState
		failedProvisioningState = doc.OpenShiftCluster.Properties.FailedProvisioningState

		if backendErr == nil {
			adminUpdateError = to.StringPtr("")
		} else {
			adminUpdateError = to.StringPtr(backendErr.Error())
		}
	}

	if stop != nil {
		stop()
	}

	_, err := ocb.dbOpenShiftClusters.EndLease(ctx, doc.Key, provisioningState, failedProvisioningState, adminUpdateError)
	return err
}

func (ocb *openShiftClusterBackend) emitMetrics(doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState) {
	if doc.CorrelationData == nil {
		return
	}

	duration := time.Since(doc.CorrelationData.RequestTime).Milliseconds()

	ocb.m.EmitGauge("backend.openshiftcluster.duration", duration, map[string]string{
		"oldProvisioningState": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		"newProvisioningState": string(provisioningState),
	})

	ocb.m.EmitGauge("backend.openshiftcluster.count", 1, map[string]string{
		"oldProvisioningState": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		"newProvisioningState": string(provisioningState),
	})
}
