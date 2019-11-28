package backend

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/backend/openshiftcluster"
)

type openShiftClusterBackend struct {
	*backend
}

func (ocb *openShiftClusterBackend) try() (bool, error) {
	doc, err := ocb.db.OpenShiftClusters.Dequeue()
	if err != nil {
		return false, err
	}
	if doc == nil {
		return false, nil
	}

	log := ocb.baseLog.WithField("resource", doc.OpenShiftCluster.ID)
	if doc.Dequeues > maxDequeueCount {
		log.Warnf("dequeued %d times, failing", doc.Dequeues)
		return true, ocb.setTerminalState(doc.OpenShiftCluster, api.ProvisioningStateFailed)
	}

	log.Print("dequeued")
	atomic.AddInt32(&ocb.workers, 1)
	go func() {
		defer func() {
			atomic.AddInt32(&ocb.workers, -1)
			ocb.cond.Signal()
		}()

		t := time.Now()

		err := ocb.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}

		log.WithField("durationMs", int(time.Now().Sub(t)/time.Millisecond)).Print("done")
	}()

	return true, nil
}

func (ocb *openShiftClusterBackend) handle(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	stop := ocb.heartbeat(log, doc.OpenShiftCluster)
	defer stop()

	m, err := openshiftcluster.NewManager(log, ocb.db.OpenShiftClusters, ocb.authorizer, doc.OpenShiftCluster, ocb.domain)
	if err != nil {
		log.Error(err)
		return ocb.setTerminalState(doc.OpenShiftCluster, api.ProvisioningStateFailed)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateUpdating:
		log.Print("updating")
		err = m.Update(ctx)
	case api.ProvisioningStateDeleting:
		log.Print("deleting")
		err = m.Delete(ctx)
	}

	stop()

	if err != nil {
		log.Error(err)
		return ocb.setTerminalState(doc.OpenShiftCluster, api.ProvisioningStateFailed)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateUpdating:
		return ocb.setTerminalState(doc.OpenShiftCluster, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateDeleting:
		return ocb.db.OpenShiftClusters.Delete(doc)

	default:
		return fmt.Errorf("unexpected state %q", doc.OpenShiftCluster.Properties.ProvisioningState)
	}
}

func (ocb *openShiftClusterBackend) heartbeat(log *logrus.Entry, oc *api.OpenShiftCluster) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := ocb.db.OpenShiftClusters.Lease(oc.Key)
			if err != nil {
				log.Error(err)
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

func (ocb *openShiftClusterBackend) setTerminalState(oc *api.OpenShiftCluster, state api.ProvisioningState) error {
	var failedOperation api.FailedOperation
	switch {
	case state == api.ProvisioningStateFailed && oc.Properties.Installation != nil:
		failedOperation = api.FailedOperationInstall
	case state == api.ProvisioningStateFailed && oc.Properties.Installation == nil:
		failedOperation = api.FailedOperationUpdate
	default:
		failedOperation = api.FailedOperationNone
	}

	_, err := ocb.db.OpenShiftClusters.Patch(oc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisioningState = state
		doc.OpenShiftCluster.Properties.FailedOperation = failedOperation

		doc.LeaseOwner = nil
		doc.LeaseExpires = 0
		doc.Dequeues = 0
		return nil
	})
	return err
}
