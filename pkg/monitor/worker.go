package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/buckets"
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

type clusterChangeFeedResponder struct {
	now        func() time.Time
	log        *logrus.Entry
	workerPool buckets.BucketWorkerPool[*api.OpenShiftClusterDocument]

	lastChangefeedProcessed  atomic.Value // time.Time
	lastChangefeedDataUpdate atomic.Value // time.Time
	lastBucketUpdate         atomic.Value // time.Time
}

func NewClusterChangefeedResponder(log *logrus.Entry, now func() time.Time, workerFunc func(<-chan struct{}, string)) *clusterChangeFeedResponder {
	return &clusterChangeFeedResponder{
		now:        now,
		log:        log,
		workerPool: buckets.NewBucketWorkerPool[*api.OpenShiftClusterDocument](log, workerFunc),
	}
}

var _ changefeed.ChangefeedConsumer[*api.OpenShiftClusterDocument] = &clusterChangeFeedResponder{}

// Update the buckets that we want to pay attention to.
func (c *clusterChangeFeedResponder) UpdateBuckets(buckets []int) {
	c.workerPool.SetBuckets(buckets)
	c.lastBucketUpdate.Store(c.now())
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
		c.workerPool.DeleteDoc(doc)
	default:
		c.workerPool.UpsertDoc(stripUnusedFields(doc))
	}
}

func (c *clusterChangeFeedResponder) GetCacheSize() int {
	return c.workerPool.CacheSize()
}

func (c *clusterChangeFeedResponder) GetLastBucketUpdate() (time.Time, bool) {
	t, ok := c.lastBucketUpdate.Load().(time.Time)
	return t, ok
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
	return c.workerPool.Doc(id)
}

func (c *clusterChangeFeedResponder) OnAllPendingProcessed(didUpdate bool) {
	now := c.now()
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

	log := utillog.EnrichWithResourceID(mon.baseLog, id)
	r, err := azure.ParseResourceID(id)
	if err != nil {
		log.Error(err)
		return
	}
	// Wait for a randomised delay before starting
	delay := time.Second * time.Duration(mon.workerMaxStartupDelay.Seconds()*rand.Float64())
	log.Debugf("starting worker for %s in %s...", id, delay.String())
	time.Sleep(delay)
	log.Debugf("starting monitoring for %s", id)

	nsgMonitoringTicker := time.NewTicker(nsgMonitoringFrequency)
	defer nsgMonitoringTicker.Stop()
	subscriptionStateLoggingTicker := time.NewTicker(subscriptionStateLogFrequency)
	defer subscriptionStateLoggingTicker.Stop()
	t := time.NewTicker(mon.interval)
	defer t.Stop()

	h := mon.env.Now().Hour()

out:
	for {
		select {
		case <-t.C:
			func() {
				mon.workerCount.Add(1)
				mon.m.EmitGauge("monitor.workers.active.count", int64(mon.workerCount.Load()), nil)
				defer func() {
					mon.workerCount.Add(-1)
					mon.m.EmitGauge("monitor.workers.active.count", int64(mon.workerCount.Load()), nil)
				}()

				doc, ok := mon.clusters.GetCluster(id)
				if !ok {
					return
				}
				subID := strings.ToLower(r.SubscriptionID)
				sub, subok := mon.subs.GetSubscription(subID)

				if !subok {
					select {
					case <-subscriptionStateLoggingTicker.C:
						// The changefeed filters out subscriptions in invalid states
						log.Warningf("Skipped monitoring cluster %s because its subscription is in an invalid state", doc.OpenShiftCluster.ID)

					default:
					}
					return
				}

				newh := mon.env.Now().Hour()

				// TODO: later can modify here to poll once per N minutes and re-issue
				// cached metrics in the remaining minutes
				mon.workOne(context.Background(), log, doc, subID, sub.TenantID, newh != h, nsgMonitoringTicker)

				h = newh
			}()
		case <-stop:
			break out
		}
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
