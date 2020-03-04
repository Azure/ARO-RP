package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/cluster"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

// listBuckets reads our bucket allocation from the master
func (mon *monitor) listBuckets(ctx context.Context) error {
	buckets, err := mon.db.Monitors.ListBuckets(ctx)

	mon.mu.Lock()
	defer mon.mu.Unlock()

	if len(buckets) != len(mon.buckets) {
		mon.baseLog.Printf("servicing %d buckets", len(buckets))
	}

	mon.buckets = map[int]struct{}{}

	if err != nil {
		return err
	}

	for _, i := range buckets {
		mon.buckets[i] = struct{}{}
	}

	return nil
}

// changefeed tracks the OpenShiftClusters change feed and keeps mon.docs
// up-to-date.  We don't monitor clusters in Creating state, hence we don't add
// them to mon.docs.  We also don't monitor clusters in Deleting state; when
// this state is reached we delete from mon.docs
func (mon *monitor) changefeed(ctx context.Context, baseLog *logrus.Entry, stop <-chan struct{}) {
	defer recover.Panic(baseLog)

	i := mon.db.OpenShiftClusters.ChangeFeed()

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		for {
			docs, err := i.Next(ctx)
			if err != nil {
				baseLog.Error(err)
				break
			}
			if docs == nil {
				break
			}

			for _, doc := range docs.OpenShiftClusterDocuments {
				r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
				if err != nil {
					baseLog.Error(err)
					continue
				}

				log := baseLog.WithFields(logrus.Fields{
					"resource_id":     doc.OpenShiftCluster.ID,
					"subscription_id": r.SubscriptionID,
					"resource_group":  r.ResourceGroup,
					"resource_name":   r.ResourceName,
				})

				ps := doc.OpenShiftCluster.Properties.ProvisioningState
				fps := doc.OpenShiftCluster.Properties.FailedProvisioningState
				log.Debugf("cluster in provisioningState %s", ps)
				switch {
				case ps == api.ProvisioningStateCreating,
					ps == api.ProvisioningStateDeleting,
					ps == api.ProvisioningStateFailed &&
						(fps == api.ProvisioningStateCreating ||
							fps == api.ProvisioningStateDeleting):
					mon.docs.Delete(doc.ID)
				default:
					// TODO: improve memory usage by storing a subset of doc in mon.docs
					mon.docs.Store(doc.ID, doc)
				}
			}
		}

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}

// schedule walks mon.docs and schedules work across the worker goroutines.  It
// aims for every cluster to be monitored every five minutes
func (mon *monitor) schedule(ctx context.Context, log *logrus.Entry, stop <-chan struct{}) {
	defer recover.Panic(log)

	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()

	for {
		mon.docs.Range(func(key, value interface{}) bool {
			doc := value.(*api.OpenShiftClusterDocument)

			mon.mu.Lock()
			_, found := mon.buckets[doc.Bucket]
			mon.mu.Unlock()

			if found {
				mon.ch <- doc.ID
			}

			return true
		})

		select {
		case <-t.C:
		case <-stop:
			close(mon.ch)
			return
		}
	}
}

// worker reads clusters to be monitored and monitors them
func (mon *monitor) worker(ctx context.Context, baseLog *logrus.Entry) {
	defer recover.Panic(baseLog)

	for id := range mon.ch {
		_doc, found := mon.docs.Load(id)
		if !found {
			continue
		}

		doc := _doc.(*api.OpenShiftClusterDocument)

		r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
		if err != nil {
			baseLog.Error(err)
			continue
		}

		log := baseLog.WithFields(logrus.Fields{
			"resource_id":     doc.OpenShiftCluster.ID,
			"subscription_id": r.SubscriptionID,
			"resource_group":  r.ResourceGroup,
			"resource_name":   r.ResourceName,
		})

		err = mon.workOne(ctx, log, doc)
		if err != nil {
			log.Error(err)
		}
	}
}

// workOne checks the API server health of a cluster
func (mon *monitor) workOne(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	log.Debug("monitoring")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	c, err := cluster.NewMonitor(ctx, mon.env, log, doc.OpenShiftCluster, mon.clusterm)
	if err != nil {
		return err
	}

	return c.Monitor(ctx)
}
