package backend

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/backend/openshiftcluster"
	"github.com/jim-minter/rp/pkg/util/recover"
)

type openShiftClusterBackend struct {
	*backend
}

// try tries to dequeue an OpenShiftClusterDocument for work, and works it on a
// new goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (ocb *openShiftClusterBackend) try() (bool, error) {
	doc, err := ocb.db.OpenShiftClusters.Dequeue()
	if err != nil || doc == nil {
		return false, err
	}

	log := ocb.baseLog.WithField("resource", doc.OpenShiftCluster.ID)
	if doc.Dequeues > maxDequeueCount {
		log.Warnf("dequeued %d times, failing", doc.Dequeues)
		return true, ocb.endLease(nil, doc, api.ProvisioningStateFailed)
	}

	log.Print("dequeued")
	atomic.AddInt32(&ocb.workers, 1)
	go func() {
		defer recover.Panic(log)

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
	stop := ocb.heartbeat(log, doc)
	defer stop()

	m, err := openshiftcluster.NewManager(log, ocb.env, ocb.db.OpenShiftClusters, ocb.fpAuthorizer, doc)
	if err != nil {
		log.Error(err)
		return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateCreating:
		log.Print("creating")

		err = m.Create(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		return ocb.endLease(stop, doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateUpdating:
		log.Print("updating")

		err = m.Update(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		return ocb.endLease(stop, doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateDeleting:
		log.Print("deleting")
		err = m.Delete(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		stop()
		return ocb.db.OpenShiftClusters.Delete(doc)

	}

	return fmt.Errorf("unexpected provisioningState %q", doc.OpenShiftCluster.Properties.ProvisioningState)
}

func (ocb *openShiftClusterBackend) heartbeat(log *logrus.Entry, doc *api.OpenShiftClusterDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := ocb.db.OpenShiftClusters.Lease(doc.Key)
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

func (ocb *openShiftClusterBackend) endLease(stop func(), doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState) error {
	var failedProvisioningState api.ProvisioningState
	if provisioningState == api.ProvisioningStateFailed {
		failedProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	}

	if stop != nil {
		stop()
	}

	_, err := ocb.db.OpenShiftClusters.EndLease(doc.Key, provisioningState, failedProvisioningState)
	return err
}
