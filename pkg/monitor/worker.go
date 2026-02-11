package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
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

// listBuckets reads our bucket allocation from the master
func (mon *monitor) listBuckets(ctx context.Context) error {
	dbMonitors, err := mon.dbGroup.Monitors()
	if err != nil {
		return err
	}

	buckets, err := dbMonitors.ListBuckets(ctx)

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

	return err
}

type clusterChangeFeedResponder struct {
	mon *monitor
}

func (c *clusterChangeFeedResponder) Lock() {
	c.mon.mu.Lock()
}

func (c *clusterChangeFeedResponder) Unlock() {
	c.mon.mu.Unlock()
}

func (c *clusterChangeFeedResponder) OnDoc(doc *api.OpenShiftClusterDocument) {
	ps := doc.OpenShiftCluster.Properties.ProvisioningState
	fps := doc.OpenShiftCluster.Properties.FailedProvisioningState

	switch {
	case ps == api.ProvisioningStateCreating,
		ps == api.ProvisioningStateDeleting,
		ps == api.ProvisioningStateFailed &&
			(fps == api.ProvisioningStateCreating ||
				fps == api.ProvisioningStateDeleting):
		c.mon.deleteDoc(doc)
	default:
		c.mon.upsertDoc(doc)
	}
}

func (c *clusterChangeFeedResponder) OnAllPendingProcessed() {
	c.mon.lastClusterChangefeed.Store(time.Now())
}

// changefeedMetrics emits metrics tracking the size of the changefeed caches.
func (mon *monitor) changefeedMetrics(stop <-chan struct{}) {
	defer recover.Panic(mon.baseLog)

	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		mon.m.EmitGauge("monitor.cache.size", int64(len(mon.docs)), map[string]string{"cache": "openshiftclusters"})
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
		mon.mu.RLock()
		v := mon.docs[id]
		mon.mu.RUnlock()

		if v == nil {
			return
		}

		log = utillog.EnrichWithResourceID(log, v.doc.OpenShiftCluster.ID)

		var err error
		r, err = azure.ParseResourceID(v.doc.OpenShiftCluster.ID)
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
		mon.mu.RLock()
		v := mon.docs[id]
		mon.mu.RUnlock()
		subID := strings.ToLower(r.SubscriptionID)
		sub, subok := mon.subs.GetSubscription(subID)

		if v == nil {
			break
		}

		newh := time.Now().Hour()

		// TODO: later can modify here to poll once per N minutes and re-issue
		// cached metrics in the remaining minutes

		if subok {
			mon.workOne(context.Background(), log, v.doc, subID, sub.TenantID, newh != h, nsgMonitoringTicker)
		}

		select {
		case <-t.C:
			select {
			case <-subscriptionStateLoggingTicker.C:
				// The changefeed filters out subscriptions in invalid states
				if !subok {
					log.Warningf("Skipped monitoring cluster %s because its subscription is in an invalid state", v.doc.OpenShiftCluster.ID)
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
	allJobsDone := make(chan bool)
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
