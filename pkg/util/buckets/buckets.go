package buckets

import (
	"reflect"
	"sync"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/sirupsen/logrus"
)

type WorkerFunc func(<-chan struct{}, time.Duration, string)

type monitor struct {
	baseLog *logrus.Entry

	bucketCount int
	buckets     map[int]struct{}

	mu   *sync.RWMutex
	docs map[string]*cacheDoc

	worker WorkerFunc
}

type BucketWorker interface {
	LoadBuckets([]int)
	Balance([]string, *api.BucketServiceDocument)
	Stop()

	Doc(string) *api.OpenShiftClusterDocument
	DeleteDoc(*api.OpenShiftClusterDocument)
	UpsertDoc(*api.OpenShiftClusterDocument)
}

func NewBucketWorker(log *logrus.Entry, worker WorkerFunc, mu *sync.RWMutex) *monitor {
	return &monitor{
		baseLog: log,

		worker: worker,
		docs:   map[string]*cacheDoc{},

		buckets:     map[int]struct{}{},
		bucketCount: bucket.Buckets,

		mu: mu,
	}

}

// LoadBuckets is called with the bucket allocation from the controller
func (mon *monitor) LoadBuckets(buckets []int) {
	mon.mu.Lock()
	defer mon.mu.Unlock()

	oldBuckets := mon.buckets
	mon.buckets = make(map[int]struct{}, len(buckets))

	for _, i := range buckets {
		mon.buckets[i] = struct{}{}
	}

	if !reflect.DeepEqual(mon.buckets, oldBuckets) {
		mon.baseLog.Printf("servicing %d buckets", len(mon.buckets))
		mon.fixDocs()
	}
}

func (mon *monitor) Doc(id string) *api.OpenShiftClusterDocument {
	v := mon.docs[id]
	if v == nil {
		return nil
	}
	return v.doc
}

// balance shares out buckets over a slice of registered monitors
func (mon *monitor) Balance(monitors []string, doc *api.BucketServiceDocument) {
	// initialise doc.Buckets
	if doc.Buckets == nil {
		doc.Buckets = make([]string, 0)
	}

	// ensure len(doc.Buckets) == mon.bucketCount: this should only do
	// anything on the very first run
	if len(doc.Buckets) < mon.bucketCount {
		doc.Buckets = append(doc.Buckets, make([]string, mon.bucketCount-len(doc.Buckets))...)
	}
	if len(doc.Buckets) > mon.bucketCount { // should never happen
		doc.Buckets = doc.Buckets[:mon.bucketCount]
	}

	var unallocated []int
	m := make(map[string][]int, len(monitors)) // map of monitor to list of buckets it owns
	for _, monitor := range monitors {
		m[monitor] = nil
	}

	var target int // target number of buckets per monitor
	if len(monitors) > 0 {
		target = mon.bucketCount / len(monitors)
		if mon.bucketCount%len(monitors) != 0 {
			target++
		}
	}

	// load the current bucket allocations into the map
	for i, monitor := range doc.Buckets {
		if buckets, found := m[monitor]; found && len(buckets) < target {
			// if the current bucket is allocated to a known monitor and doesn't
			// take its number of buckets above the target, keep it there...
			m[monitor] = append(m[monitor], i)
		} else {
			// ...otherwise we'll reallocate it below
			unallocated = append(unallocated, i)
		}
	}

	// reallocate all unallocated buckets, appending to the least loaded monitor
	if len(monitors) > 0 {
		for _, i := range unallocated {
			var leastMonitor string
			for monitor := range m {
				if leastMonitor == "" ||
					len(m[monitor]) < len(m[leastMonitor]) {
					leastMonitor = monitor
				}
			}

			m[leastMonitor] = append(m[leastMonitor], i)
		}
	}

	// write the updated bucket allocations back to the document
	for _, i := range unallocated {
		doc.Buckets[i] = "" // should only happen if there are no known monitors
	}
	for monitor, buckets := range m {
		for _, i := range buckets {
			doc.Buckets[i] = monitor
		}
	}
}
