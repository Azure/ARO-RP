package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// listBuckets reads our bucket allocation from the master
func (mon *monitor) listBuckets(ctx context.Context) error {
	buckets, err := mon.dbMonitors.ListBuckets(ctx)

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

	clustersIterator := mon.dbOpenShiftClusters.ChangeFeed()
	subscriptionsIterator := mon.dbSubscriptions.ChangeFeed()

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
			mon.workOne(context.Background(), log, v.doc, newh != h)
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
func (mon *monitor) workOne(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, hourlyRun bool) {
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	restConfig, err := restconfig.RestConfig(mon.dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return
	}

	hiveRestConfig, err := mon.liveConfig.HiveRestConfig(ctx, 1)
	if err != nil {
		// TODO(hive): Update to fail once we have Hive everywhere in prod and dev
		log.Info(err)
	}

	c, err := cluster.NewMonitor(ctx, log, restConfig, doc.OpenShiftCluster, mon.clusterm, hiveRestConfig, hourlyRun)
	if err != nil {
		log.Error(err)
		return
	}

	c.Monitor(ctx)
}
