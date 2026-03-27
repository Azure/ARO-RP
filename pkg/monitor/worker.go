package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// nsgMonitoringFrequency is used for initializing NSG monitoring ticker
var nsgMonitoringFrequency = 10 * time.Minute

// subscriptionStateLogFrequency is used for initializing a ticker used to
// send log messages when a cluster's subscription state is stopping us
// from monitoring
var subscriptionStateLogFrequency = 30 * time.Minute

// changefeedBatchSize is how many items in the changefeed to fetch in each page
const changefeedBatchSize = 50

// onBuckets is called when we fetch our bucket allocation from the master
func (mon *monitor) onBuckets(buckets []int) {
	if len(buckets) == 0 {
		mon.baseLog.Error("bucket allocation contained no buckets")
		return
	}

	mon.clusters.UpdateBuckets(buckets)

	mon.lastBucketlist.Store(time.Now())
}

type clusterChangeFeedResponder struct {
	log      *logrus.Entry
	docs     *xsync.Map[string, *cacheDoc]
	bucketMu *sync.RWMutex
	buckets  map[int]struct{}

	lastChangefeedProcessed  atomic.Value // time.Time
	lastChangefeedDataUpdate atomic.Value // time.Time

	newWorker func(<-chan struct{}, string)
}

func NewClusterChangefeedResponder(log *logrus.Entry, workerFunc func(<-chan struct{}, string)) *clusterChangeFeedResponder {
	return &clusterChangeFeedResponder{
		log:      log,
		docs:     xsync.NewMap[string, *cacheDoc](),
		bucketMu: &sync.RWMutex{},
		buckets:  map[int]struct{}{},

		newWorker: workerFunc,
	}
}

var _ changefeed.ChangefeedConsumer[*api.OpenShiftClusterDocument] = &clusterChangeFeedResponder{}

// Update the buckets that we want to pay attention to.
func (c *clusterChangeFeedResponder) UpdateBuckets(buckets []int) {
	c.bucketMu.Lock()
	defer c.bucketMu.Unlock()
	oldBuckets := c.buckets
	c.buckets = make(map[int]struct{}, len(buckets))

	for _, i := range buckets {
		c.buckets[i] = struct{}{}
	}

	if !reflect.DeepEqual(c.buckets, oldBuckets) {
		c.log.Printf("servicing %d buckets", len(c.buckets))
		c.fixDocs()
	}
}

// we don't use a mutex internally, we use a xsync.Map, so Lock/Unlock are
// no-ops
func (c *clusterChangeFeedResponder) Lock()   {}
func (c *clusterChangeFeedResponder) Unlock() {}

func (c *clusterChangeFeedResponder) OnDoc(doc *api.OpenShiftClusterDocument) {
	ps := doc.OpenShiftCluster.Properties.ProvisioningState
	fps := doc.OpenShiftCluster.Properties.FailedProvisioningState

	switch {
	case ps == api.ProvisioningStateCreating,
		ps == api.ProvisioningStateDeleting,
		ps == api.ProvisioningStateFailed &&
			(fps == api.ProvisioningStateCreating ||
				fps == api.ProvisioningStateDeleting):
		// If the provisioning state is creating/deleting or failed during
		// creating/deleting, remove the cluster from monitoring. A fully
		// created cluster will later trigger the changefeed with a 'succeeded'
		// state, while deleting documents will not appear in the changefeed
		// once they are actually deleted, so we need to remove them when they
		// start deletion.
		//
		// If the cluster is already not monitored, deleteDoc will be a no-op.
		c.deleteDoc(doc)
	default:
		c.upsertDoc(doc)
	}
}

func (c *clusterChangeFeedResponder) GetCacheSize() int {
	return c.docs.Size()
}

func (c *clusterChangeFeedResponder) GetLastProcessed() (time.Time, bool) {
	t, ok := c.lastChangefeedProcessed.Load().(time.Time)
	return t, ok
}

func (c *clusterChangeFeedResponder) GetLastDataUpdate() (time.Time, bool) {
	t, ok := c.lastChangefeedDataUpdate.Load().(time.Time)
	return t, ok
}

func (c *clusterChangeFeedResponder) GetCluster(id string) (*api.OpenShiftClusterDocument, bool) {
	v, ok := c.docs.Load(id)
	if !ok {
		return nil, ok
	}
	return v.doc, ok
}

func (c *clusterChangeFeedResponder) OnAllPendingProcessed(didUpdate bool) {
	now := time.Now()
	c.lastChangefeedProcessed.Store(now)
	if didUpdate {
		c.lastChangefeedDataUpdate.Store(now)
	}
}

// changefeedMetrics emits metrics tracking the size of the changefeed caches.
func (mon *monitor) changefeedMetrics(stop <-chan struct{}) {
	defer recover.Panic(mon.baseLog)

	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		mon.m.EmitGauge("monitor.cache.size", int64(mon.clusters.GetCacheSize()), map[string]string{"cache": "openshiftclusters"})
		mon.m.EmitGauge("monitor.cache.size", int64(mon.subs.GetCacheSize()), map[string]string{"cache": "subscriptions"})

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}

// worker reads clusters to be monitored and monitors them
func (mon *monitor) worker(stop <-chan struct{}, id string) {
	defer recover.Panic(mon.baseLog)

	time.Sleep(mon.delay)

	var r azure.Resource

	log := mon.baseLog
	{
		doc, ok := mon.clusters.GetCluster(id)
		if !ok {
			return
		}

		log = utillog.EnrichWithResourceID(log, doc.OpenShiftCluster.ID)

		var err error
		r, err = azure.ParseResourceID(doc.OpenShiftCluster.ID)
		if err != nil {
			log.Error(err)
			return
		}
	}

	log.Debug("starting monitoring")

	nsgMonitoringTicker := time.NewTicker(nsgMonitoringFrequency)
	defer nsgMonitoringTicker.Stop()
	subscriptionStateLoggingTicker := time.NewTicker(subscriptionStateLogFrequency)
	defer subscriptionStateLoggingTicker.Stop()
	t := time.NewTicker(mon.interval)
	defer t.Stop()

	h := time.Now().Hour()

out:
	for {
		doc, ok := mon.clusters.GetCluster(id)
		if !ok {
			break
		}
		subID := strings.ToLower(r.SubscriptionID)
		sub, subok := mon.subs.GetSubscription(subID)

		newh := time.Now().Hour()

		// TODO: later can modify here to poll once per N minutes and re-issue
		// cached metrics in the remaining minutes

		if subok {
			mon.workOne(context.Background(), log, doc, subID, sub.TenantID, newh != h, nsgMonitoringTicker)
		}

		select {
		case <-t.C:
			select {
			case <-subscriptionStateLoggingTicker.C:
				// The changefeed filters out subscriptions in invalid states
				if !subok {
					log.Warningf("Skipped monitoring cluster %s because its subscription is in an invalid state", doc.OpenShiftCluster.ID)
				}
			default:
			}
		case <-stop:
			break out
		}

		h = newh
	}

	log.Debug("stopping monitoring")
}

// workOne checks the API server health of a cluster
func (mon *monitor) workOne(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, subID string, tenantID string, hourlyRun bool, nsgMonTicker *time.Ticker) {
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	restConfig, err := restconfig.RestConfig(mon.dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return
	}

	dims := map[string]string{
		dimension.ResourceID:     doc.OpenShiftCluster.ID,
		dimension.Location:       doc.OpenShiftCluster.Location,
		dimension.SubscriptionID: subID,
	}

	var monitors []monitoring.Monitor

	hiveClusterManager, ok := mon.hiveClusterManagers[1]
	if !ok {
		log.Info("skipping: no hive cluster manager")
	} else {
		h, err := mon.hiveMonitorBuilder(log, doc.OpenShiftCluster, mon.clusterm, hourlyRun, hiveClusterManager)
		if err != nil {
			log.Error(err)
			mon.m.EmitGauge("monitor.hive.failedworker", 1, dims)
		} else {
			monitors = append(monitors, h)
		}
	}

	nsgMon := mon.nsgMonitorBuilder(log, doc.OpenShiftCluster, mon.env, subID, tenantID, mon.clusterm, dims, nsgMonTicker.C)

	c, err := mon.clusterMonitorBuilder(log, restConfig, doc.OpenShiftCluster, mon.env, tenantID, mon.clusterm, hourlyRun)
	if err != nil {
		log.Error(err)
		mon.m.EmitGauge("monitor.cluster.failedworker", 1, dims)
		return
	}

	monitors = append(monitors, c, nsgMon)
	allJobsDone := make(chan bool, 1)
	onPanic := func(m monitoring.Monitor) {
		// emit a failed worker metric on panic
		mon.m.EmitGauge("monitor."+m.MonitorName()+".failedworker", 1, dims)
	}
	go execute(ctx, log, allJobsDone, monitors, onPanic)

	select {
	case <-allJobsDone:
	case <-ctx.Done():
		log.Infof("The monitoring process for cluster %s has timed out.", doc.OpenShiftCluster.ID)
		mon.m.EmitGauge("monitor.main.timedout", int64(1), dims)
	}
}

func execute(ctx context.Context, log *logrus.Entry, done chan<- bool, monitors []monitoring.Monitor, onPanic func(monitoring.Monitor)) {
	var wg sync.WaitGroup

	for _, monitor := range monitors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := monitor.Monitor(ctx)
			if err != nil {
				if errors.Is(err, &monitoring.MonitorPanic{}) {
					onPanic(monitor)
				}
				log.Error(err)
			}
		}()
	}
	wg.Wait()
	done <- true
}
