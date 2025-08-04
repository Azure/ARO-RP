package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/azure/nsg"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	hivemon "github.com/Azure/ARO-RP/pkg/monitor/hive"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// nsgMonitoringFrequency is used for initializing NSG monitoring ticker
var nsgMonitoringFrequency = 10 * time.Minute

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
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		successful := true
		for {
			docs, err := clustersIterator.Next(ctx, -1)
			if err != nil {
				successful = false
				baseLog.Error(err)
				break
			}
			if docs == nil {
				break
			}

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
					// TODO: improve memory usage by storing a subset of doc in mon.docs
					mon.upsertDoc(doc)
				}
			}

			mon.mu.Unlock()
		}

		for {
			subs, err := subscriptionsIterator.Next(ctx, -1)
			if err != nil {
				successful = false
				baseLog.Error(err)
				break
			}
			if subs == nil {
				break
			}

			mon.mu.Lock()

			for _, sub := range subs.SubscriptionDocuments {
				mon.subs[sub.ID] = sub
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

// worker reads clusters to be monitored and monitors them
func (mon *monitor) worker(stop <-chan struct{}, delay time.Duration, id string) {
	defer recover.Panic(mon.baseLog)

	time.Sleep(delay)

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
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	h := time.Now().Hour()

out:
	for {
		mon.mu.RLock()
		v := mon.docs[id]
		sub := mon.subs[r.SubscriptionID]
		mon.mu.RUnlock()

		if v == nil {
			break
		}

		newh := time.Now().Hour()

		// TODO: later can modify here to poll once per N minutes and re-issue
		// cached metrics in the remaining minutes

		if sub != nil && sub.Subscription != nil && sub.Subscription.State != api.SubscriptionStateSuspended && sub.Subscription.State != api.SubscriptionStateWarned {
			mon.workOne(context.Background(), log, v.doc, sub, newh != h, nsgMonitoringTicker)
		}

		select {
		case <-t.C:
		case <-stop:
			break out
		}

		h = newh
	}

	log.Debug("stopping monitoring")
}

// workOne checks the API server health of a cluster
func (mon *monitor) workOne(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument, hourlyRun bool, nsgMonTicker *time.Ticker) {
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	restConfig, err := restconfig.RestConfig(mon.dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return
	}

	dims := map[string]string{
		dimension.ClusterResourceID: doc.OpenShiftCluster.ID,
		dimension.Location:          doc.OpenShiftCluster.Location,
		dimension.SubscriptionID:    sub.ID,
	}

	var monitors []monitoring.Monitor
	var wg sync.WaitGroup

	hiveClusterManager, ok := mon.hiveClusterManagers[1]
	if !ok {
		log.Info("skipping: no hive cluster manager")
	} else {
		h, err := hivemon.NewHiveMonitor(log, doc.OpenShiftCluster, mon.clusterm, hourlyRun, &wg, hiveClusterManager)
		if err != nil {
			log.Error(err)
			mon.m.EmitGauge("monitor.hive.failedworker", 1, map[string]string{
				"resourceId": doc.OpenShiftCluster.ID,
			})
		} else {
			monitors = append(monitors, h)
		}
	}

	nsgMon := nsg.NewMonitor(log, doc.OpenShiftCluster, mon.env, sub.ID, sub.Subscription.Properties.TenantID, mon.clusterm, dims, &wg, nsgMonTicker.C)

	c, err := cluster.NewMonitor(log, restConfig, doc.OpenShiftCluster, doc, mon.env, sub.Subscription.Properties.TenantID, mon.clusterm, hourlyRun, &wg)
	if err != nil {
		log.Error(err)
		mon.m.EmitGauge("monitor.cluster.failedworker", 1, map[string]string{
			"resourceId": doc.OpenShiftCluster.ID,
		})
		return
	}

	monitors = append(monitors, c, nsgMon)
	allJobsDone := make(chan bool)
	go execute(ctx, allJobsDone, &wg, monitors)

	select {
	case <-allJobsDone:
	case <-ctx.Done():
		log.Infof("The monitoring process for cluster %s has timed out.", doc.OpenShiftCluster.ID)
		mon.m.EmitGauge("monitor.main.timedout", int64(1), dims)
	}
}

func execute(ctx context.Context, done chan<- bool, wg *sync.WaitGroup, monitors []monitoring.Monitor) {
	for _, monitor := range monitors {
		wg.Add(1)
		go monitor.Monitor(ctx)
	}
	wg.Wait()
	done <- true
}
