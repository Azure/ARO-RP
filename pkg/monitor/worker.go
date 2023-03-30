package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// This function will continue to run until such time as it has a config to add to the global Hive shard map
// Note that because the mon.hiveShardConfigs[shard] is set to `nil` when its created, the cluster
// monitors will simply ignore Hive stats until this function populates the config
func (mon *monitor) populateHiveShardRestConfig(ctx context.Context, shard int) {
	var hiveRestConfig *rest.Config
	var err error

	for {
		hiveRestConfig, err = mon.liveConfig.HiveRestConfig(ctx, shard)
		if hiveRestConfig != nil {
			mon.setHiveShardConfig(shard, hiveRestConfig)
			return
		}

		mon.baseLog.Warn(fmt.Sprintf("error fetching Hive kubeconfig for shard %d", shard))
		if err != nil {
			mon.baseLog.Error(err.Error())
		}

		mon.baseLog.Info("pausing for a minute before retrying...")
		time.Sleep(60 * time.Second)
	}
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
					// in the future we will have the shard index set on the api.OpenShiftClusterDocument
					// but for now we simply select Hive (AKS) shard 1
					// e.g. shard := mon.hiveShardConfigs[doc.shardIndex]
					shard := 1

					_, exists := mon.getHiveShardConfig(shard)
					if !exists {
						// set this to `nil` so cluster monitors will ignore it until its populated with config
						mon.setHiveShardConfig(shard, nil)
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

	// once sharding is implemented, we will have the shard set on the api.OpenShiftClusterDocument
	// e.g. shard := mon.hiveShardConfigs[doc.shardIndex]
	shard := 1
	hiveRestConfig, exists := mon.getHiveShardConfig(shard)
	if !exists {
		log.Warnf("no hiveShardConfigs set for shard %d", shard)
	}

	c, err := cluster.NewMonitor(log, restConfig, doc.OpenShiftCluster, mon.clusterm, hiveRestConfig, hourlyRun)
	if err != nil {
		log.Error(err)
		return
	}

	c.Monitor(ctx)
}
