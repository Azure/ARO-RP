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

// changefeed tracks the OpenShiftClusters change feed and keeps mon.docs
// up-to-date.  We don't monitor clusters in Creating state, hence we don't add
// them to mon.docs.  We also don't monitor clusters in Deleting state; when
// this state is reached we delete from mon.docs
func (mon *monitor) changefeed(ctx context.Context, baseLog *logrus.Entry, stop <-chan struct{}) {
	defer recover.Panic(baseLog)

	dbOpenShiftClusters, err := mon.dbGroup.OpenShiftClusters()
	if err != nil {
		baseLog.Error(err)
		panic(err)
	}

	dbSubscriptions, err := mon.dbGroup.Subscriptions()
	if err != nil {
		baseLog.Error(err)
		panic(err)
	}

	clustersIterator := dbOpenShiftClusters.ChangeFeed()
	subscriptionsIterator := dbSubscriptions.ChangeFeed()

	// Align this time with the deletion mechanism.
	// Go to docs/monitoring.md for the details.
	t := time.NewTicker(mon.changefeedInterval)
	defer t.Stop()

	for {
		successful := true
		for {
			docs, err := clustersIterator.Next(ctx, changefeedBatchSize)
			if err != nil {
				successful = false
				baseLog.Error(err)
				break
			}
			if docs == nil {
				break
			}

			baseLog.Debugf("openshiftclusters changefeed page was %d docs", docs.Count)

			mon.mu.Lock()

			for _, doc := range docs.OpenShiftClusterDocuments {
				ps := doc.OpenShiftCluster.Properties.ProvisioningState
				fps := doc.OpenShiftCluster.Properties.FailedProvisioningState

				switch {
				case ps == api.ProvisioningStateCreating,
					ps == api.ProvisioningStateDeleting,
					ps == api.ProvisioningStateFailed &&
						(fps == api.ProvisioningStateCreating ||
							fps == api.ProvisioningStateDeleting):
					mon.deleteDoc(doc)
				default:
					mon.upsertDoc(doc)
				}
			}

			mon.mu.Unlock()
		}

		for {
			subs, err := subscriptionsIterator.Next(ctx, changefeedBatchSize)
			if err != nil {
				successful = false
				baseLog.Error(err)
				break
			}
			if subs == nil {
				break
			}

			baseLog.Debugf("subscriptions changefeed page was %d docs", subs.Count)

			mon.mu.Lock()

			for _, sub := range subs.SubscriptionDocuments {
				id := strings.ToLower(sub.ID)

				// Don't keep subscriptions that are restricted, warned, or are
				// being deleted from our db
				if sub.Subscription.State == api.SubscriptionStateSuspended ||
					sub.Subscription.State == api.SubscriptionStateWarned ||
					sub.Subscription.State == api.SubscriptionStateDeleted {
					// delete is a no-op if it doesn't exist
					delete(mon.subs, id)
					continue
				}
				c, ok := mon.subs[id]
				if ok {
					// update this as subscription might have moved tenants
					c.TenantID = strings.ToLower(sub.Subscription.Properties.TenantID)
				} else {
					mon.subs[id] = &subscriptionInfo{
						TenantID: strings.ToLower(sub.Subscription.Properties.TenantID),
					}
				}
			}

			mon.mu.Unlock()
		}

		if successful {
			mon.lastChangefeed.Store(time.Now())
		}

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}

// changefeedMetrics emits metrics tracking the size of the changefeed caches.
func (mon *monitor) changefeedMetrics(stop <-chan struct{}) {
	defer recover.Panic(mon.baseLog)

	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		mon.m.EmitGauge("monitor.cache.size", int64(len(mon.docs)), map[string]string{"cache": "openshiftclusters"})
		mon.m.EmitGauge("monitor.cache.size", int64(len(mon.subs)), map[string]string{"cache": "subscriptions"})

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
		subID := strings.ToLower(r.SubscriptionID)
		sub := mon.subs[subID]
		mon.mu.RUnlock()

		if v == nil {
			break
		}

		newh := time.Now().Hour()

		// TODO: later can modify here to poll once per N minutes and re-issue
		// cached metrics in the remaining minutes

		if sub != nil {
			mon.workOne(context.Background(), log, v.doc, subID, sub.TenantID, newh != h, nsgMonitoringTicker)
		}

		select {
		case <-t.C:
			select {
			case <-subscriptionStateLoggingTicker.C:
				// The changefeed filters out subscriptions in invalid states
				if sub == nil {
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
	monitorCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
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
	defer closeMonitors(monitors)

	allJobsDone := make(chan bool, 1)
	onPanic := func(m monitoring.Monitor) {
		mon.m.EmitGauge("monitor."+m.MonitorName()+".failedworker", 1, dims)
	}

	go execute(monitorCtx, log, allJobsDone, monitors, onPanic)

	select {
	case <-allJobsDone:
		return
	case <-monitorCtx.Done():
		log.Infof("The monitoring process for cluster %s has timed out.", doc.OpenShiftCluster.ID)
		mon.m.EmitGauge("monitor.main.timedout", int64(1), dims)
	}

	// Wait for graceful completion before cleanup
	gracePeriod := time.NewTimer(10 * time.Second)
	defer gracePeriod.Stop()

	select {
	case <-allJobsDone:
	case <-gracePeriod.C:
		mon.m.EmitGauge("monitor.main.forcedcleanup", int64(1), dims)
	}
}

func closeMonitors(monitors []monitoring.Monitor) {
	for _, m := range monitors {
		if closeable, ok := m.(monitoring.Closeable); ok {
			closeable.Close()
		}
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
