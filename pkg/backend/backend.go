package backend

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type backend struct {
	baseLog    *logrus.Entry
	db         database.OpenShiftClusters
	authorizer autorest.Authorizer

	mu       sync.Mutex
	cond     *sync.Cond
	workers  int32
	stopping atomic.Value

	domain string
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewBackend returns a new runnable backend
func NewBackend(log *logrus.Entry, authorizer autorest.Authorizer, db database.OpenShiftClusters, domain string) Runnable {
	b := &backend{
		baseLog:    log,
		db:         db,
		authorizer: authorizer,
		domain:     domain,
	}

	b.cond = sync.NewCond(&b.mu)
	b.stopping.Store(false)

	return b
}

func (b *backend) Run(stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	go func() {
		<-stop
		b.baseLog.Print("stopping")
		b.stopping.Store(true)
		b.cond.Signal()
	}()

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) == maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		}

		doc, err := b.db.Dequeue()
		if err != nil || doc == nil {
			if err != nil {
				b.baseLog.Error(err)
			}
			<-t.C
			continue
		}

		log := b.baseLog.WithField("resource", doc.OpenShiftCluster.ID)
		if doc.Dequeues > maxDequeueCount {
			log.Warnf("dequeued %d times, failing", doc.Dequeues)
			err = b.setTerminalState(doc, api.ProvisioningStateFailed)
			if err != nil {
				log.Error(err)
			}

		} else {
			log.Print("dequeued")
			go func() {
				atomic.AddInt32(&b.workers, 1)

				defer func() {
					atomic.AddInt32(&b.workers, -1)
					b.cond.Signal()
				}()

				t := time.Now()

				err := b.handle(context.Background(), log, doc)
				if err != nil {
					log.Error(err)
				}

				log.WithField("durationMs", int(time.Now().Sub(t)/time.Millisecond)).Print("done")
			}()
		}
	}
}

func (b *backend) handle(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	stop := b.heartbeat(log, doc)
	defer stop()

	var err error
	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateUpdating:
		log.Print("updating")
		err = b.update(ctx, log, doc)
	case api.ProvisioningStateDeleting:
		log.Print("deleting")
		err = b.delete(ctx, log, doc)
	}

	stop()

	if err != nil {
		log.Error(err)
		return b.setTerminalState(doc, api.ProvisioningStateFailed)
	}

	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateUpdating:
		return b.setTerminalState(doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateDeleting:
		return b.db.Delete(doc.OpenShiftCluster.ID)

	default:
		return fmt.Errorf("unexpected state %q", doc.OpenShiftCluster.Properties.ProvisioningState)
	}
}

func (b *backend) heartbeat(log *logrus.Entry, doc *api.OpenShiftClusterDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			_, err := b.db.Lease(doc.OpenShiftCluster.ID)
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

func (b *backend) setTerminalState(doc *api.OpenShiftClusterDocument, state api.ProvisioningState) error {
	_, err := b.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
		doc.LeaseOwner = nil
		doc.LeaseExpires = 0
		doc.Dequeues = 0
		doc.OpenShiftCluster.Properties.ProvisioningState = state
		return nil
	})
	return err
}
