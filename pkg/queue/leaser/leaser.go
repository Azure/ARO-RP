package leaser

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

type Leaser interface {
	HoldLease() bool
}

type leaser struct {
	log *logrus.Entry
	lc  cosmosdb.LeaseDocumentClient

	uuid      uuid.UUID
	leaseid   string
	holdLease bool

	ticker *time.Ticker

	leaseLength time.Duration
	validUntil  time.Time
}

func NewLeaser(log *logrus.Entry, dbc cosmosdb.DatabaseClient, dbid, collid, leaseid string, refresh, leaseLength time.Duration) Leaser {
	collc := cosmosdb.NewCollectionClient(dbc, dbid)

	return &leaser{
		log:         log,
		lc:          cosmosdb.NewLeaseDocumentClient(collc, collid),
		uuid:        uuid.NewV4(),
		leaseid:     leaseid,
		ticker:      time.NewTicker(refresh),
		leaseLength: leaseLength,
	}
}

func (l *leaser) refresh() error {
	doc, err := l.lc.Get(l.leaseid, l.leaseid)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return err
	}

	validUntil := time.Now().Add(l.leaseLength)

	if doc != nil && uuid.Equal(doc.Holder, l.uuid) {
		_, err = l.lc.Replace(l.leaseid, doc)
		switch {
		case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
			doc = nil
		case err != nil:
			return err
		default:
			l.validUntil = validUntil
			return nil
		}
	}

	if doc == nil {
		_, err = l.lc.Create(l.leaseid, &api.LeaseDocument{
			ID:     l.leaseid,
			TTL:    int(l.leaseLength / time.Second),
			Holder: l.uuid,
		})

		switch {
		case cosmosdb.IsErrorStatusCode(err, http.StatusConflict):
			return nil
		case err != nil:
			return err
		default:
			l.validUntil = validUntil
			return nil
		}
	}

	return nil

}

func (l *leaser) HoldLease() bool {
	oldHoldLease := l.holdLease

	select {
	case <-l.ticker.C:
		err := l.refresh()
		if err != nil {
			l.log.Error(err)
		}
	default:
	}

	l.holdLease = time.Now().Before(l.validUntil)

	if oldHoldLease != l.holdLease {
		if oldHoldLease {
			l.log.Printf("lost %s lease", l.leaseid)
		} else {
			l.log.Printf("gained %s lease", l.leaseid)
		}
	}

	return l.holdLease
}
