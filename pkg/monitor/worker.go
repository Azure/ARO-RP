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

// This function will recurse until such time as it has a config to add to the global Hive shard map
// Note that because the mon.hiveShardConfigs[shard] is set to `nil` when its created the cluster
// monitors will simply ignore Hive stats until this function populates the config
func (mon *monitor) populateHiveShardRestConfig(ctx context.Context, shard int) {
	hiveRestConfig, err := mon.liveConfig.HiveRestConfig(ctx, shard)
	if err != nil {
		mon.baseLog.Printf("error fetching Hive kubeconfig for shard %d: %s", shard, err.Error())
		mon.baseLog.Printf("pausing for a minute before retrying...")
		time.Sleep(time.Duration(60 * time.Second))
		mon.populateHiveShardRestConfig(ctx, shard)
		return
	}
	mon.shardMutex.Lock()
	mon.hiveShardConfigs[shard] = hiveRestConfig
	mon.shardMutex.Unlock()
}

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
					// check if we have a Hive shard config and if not start the recursive auth call
					// in the future we will have the shard index set on the api.OpenShiftClusterDocument
					// but for now we simply select Hive (AKS) shard 1
					// e.g. shard := mon.hiveShardConfigs[doc.shardIndex]
					shard := 1

					mon.shardMutex.RLock()
					_, exists := mon.hiveShardConfigs[shard]
					mon.shardMutex.RUnlock()
					if !exists {
						// set this to `nil` so cluster monitors will ignore it until its populated with config
						mon.hiveShardConfigs[shard] = nil
						go mon.populateHiveShardRestConfig(ctx, shard)
					}

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

	// in the future we will have the shard index set on the api.OpenShiftClusterDocument but for
	// now we simply select Hive (AKS) shard 1 because we only have one and sharding is yet to come
	// e.g. shard := mon.hiveShardConfigs[doc.shardIndex]
	shard := 1
	mon.shardMutex.RLock()
	hiveRestConfig, exists := mon.hiveShardConfigs[shard]
	mon.shardMutex.RUnlock()
	if !exists {
		log.Error("no hiveShardConfigs set for shard %d", shard)
	}

	c, err := cluster.NewMonitor(ctx, log, restConfig, doc.OpenShiftCluster, mon.clusterm, hiveRestConfig, hourlyRun)
	if err != nil {
		log.Error(err)
		return
	}

	c.Monitor(ctx)
}
