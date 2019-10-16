package forwarder

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/queue"
	"github.com/jim-minter/rp/pkg/queue/leaser"
)

type forwarder struct {
	baseLog *logrus.Entry
	q       queue.Queue
	db      database.OpenShiftClusters
	l       leaser.Leaser
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewForwarder returns a new runnable forwarder
func NewForwarder(log *logrus.Entry, q queue.Queue, db database.OpenShiftClusters, l leaser.Leaser) Runnable {
	return &forwarder{
		baseLog: log,
		q:       q,
		db:      db,
		l:       l,
	}
}

func (f *forwarder) Run(stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		if f.l.HoldLease() {
			err := f.runOnce()
			if err != nil {
				f.baseLog.Error(err)
			}
		}

		select {
		case <-t.C:
		case <-stop:
			f.baseLog.Print("stopping")
			return
		}
	}
}

func (f *forwarder) runOnce() error {
	i := f.db.ListUnqueued()

	for {
		docs, err := i.Next()
		if err != nil {
			return err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			log := f.baseLog.WithField("resource", doc.OpenShiftCluster.ID)
			err = f.q.Put(doc.OpenShiftCluster.ID)
			if err != nil {
				return err
			}
			log.Print("enqueued")

			doc, err = f.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) (err error) {
				doc.Unqueued = false
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
