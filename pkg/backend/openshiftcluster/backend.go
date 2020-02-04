package openshiftcluster

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

type OpenShiftBackend struct {
	env env.Interface
	db  *database.Database
	m   metrics.Interface
}

func New(env env.Interface, db *database.Database, m metrics.Interface) *OpenShiftBackend {
	return &OpenShiftBackend{
		env: env,
		db:  db,
		m:   m,
	}
}

func (ocb *OpenShiftBackend) Dequeue() func(context.Context) (interface{}, error) {
	return ocb.db.OpenShiftClusters.DequeueRaw
}

// Handle is responsible for handling backend operation all related operations
// to this backend
func (ocb *OpenShiftBackend) Handle(ctx context.Context, baseLog *logrus.Entry, docRaw interface{}) error {
	t := time.Now()
	doc := docRaw.(*api.OpenShiftClusterDocument)
	log := baseLog.WithField("resource", doc.OpenShiftCluster.ID)

	defer func() {
		ocb.m.EmitFloat("backend.openshiftcluster.duration", time.Now().Sub(t).Seconds(), map[string]string{
			"state": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		})

		ocb.m.EmitGauge("backend.openshiftcluster.count", 1, map[string]string{
			"state": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		})
	}()

	if doc.Dequeues > maxDequeueCount {
		log.Errorf("dequeued %d times, failing", doc.Dequeues)
		return ocb.endLease(ctx, nil, doc, api.ProvisioningStateFailed)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := ocb.heartbeat(ctx, cancel, log, doc)
	defer stop()

	m, err := NewManager(log, ocb.env, ocb.db.OpenShiftClusters, doc)
	if err != nil {
		log.Error(err)
		return ocb.endLease(ctx, stop, doc, api.ProvisioningStateFailed)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateCreating:
		log.Print("creating")

		ok, err := m.Create(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(ctx, stop, doc, api.ProvisioningStateFailed)
		}
		if ok {
			return ocb.endLease(ctx, stop, doc, api.ProvisioningStateSucceeded)
		}
		stop()
		return nil

	case api.ProvisioningStateUpdating:
		log.Print("updating")

		err = m.Update(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(ctx, stop, doc, api.ProvisioningStateFailed)
		}

		return ocb.endLease(ctx, stop, doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateDeleting:
		log.Print("deleting")

		err = m.Delete(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(ctx, stop, doc, api.ProvisioningStateFailed)
		}

		err = ocb.updateAsyncOperation(ctx, doc.AsyncOperationID, nil, api.ProvisioningStateSucceeded, "")
		if err != nil {
			log.Error(err)
			return ocb.endLease(ctx, stop, doc, api.ProvisioningStateFailed)
		}

		stop()

		return ocb.db.OpenShiftClusters.Delete(ctx, doc)
	}

	return fmt.Errorf("unexpected provisioningState %q", doc.OpenShiftCluster.Properties.ProvisioningState)
}

func (ocb *OpenShiftBackend) heartbeat(ctx context.Context, cancel context.CancelFunc, log *logrus.Entry, doc *api.OpenShiftClusterDocument) func() {
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

func (ocb *OpenShiftBackend) updateAsyncOperation(ctx context.Context, id string, oc *api.OpenShiftCluster, provisioningState, failedProvisioningState api.ProvisioningState) error {
	if id != "" {
		_, err := ocb.db.AsyncOperations.Patch(ctx, id, func(asyncdoc *api.AsyncOperationDocument) error {
			asyncdoc.AsyncOperation.ProvisioningState = provisioningState

			now := time.Now()
			asyncdoc.AsyncOperation.EndTime = &now

			if provisioningState == api.ProvisioningStateFailed {
				asyncdoc.AsyncOperation.Error = &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "Internal server error.",
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

func (ocb *OpenShiftBackend) endLease(ctx context.Context, stop func(), doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState) error {
	var failedProvisioningState api.ProvisioningState
	if provisioningState == api.ProvisioningStateFailed {
		failedProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	}

	err := ocb.updateAsyncOperation(ctx, doc.AsyncOperationID, doc.OpenShiftCluster, provisioningState, failedProvisioningState)
	if err != nil {
		return err
	}

	if stop != nil {
		stop()
	}

	_, err = ocb.db.OpenShiftClusters.EndLease(ctx, doc.Key, provisioningState, failedProvisioningState)
	return err
}
