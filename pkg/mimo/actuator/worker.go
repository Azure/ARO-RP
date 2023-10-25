package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// changefeed tracks the OpenShiftClusters change feed and keeps mon.docs
// up-to-date.  We don't monitor clusters in Creating state, hence we don't add
// them to mon.docs.  We also don't monitor clusters in Deleting state; when
// this state is reached we delete from mon.docs
func (s *service) changefeed(ctx context.Context, baseLog *logrus.Entry, stop <-chan struct{}) {
	defer recover.Panic(baseLog)

	clustersIterator := s.dbOpenShiftClusters.ChangeFeed()

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

			s.mu.Lock()

			for _, doc := range docs.OpenShiftClusterDocuments {
				ps := doc.OpenShiftCluster.Properties.ProvisioningState
				fps := doc.OpenShiftCluster.Properties.FailedProvisioningState

				switch {
				case ps == api.ProvisioningStateCreating,
					ps == api.ProvisioningStateDeleting,
					ps == api.ProvisioningStateFailed &&
						(fps == api.ProvisioningStateCreating ||
							fps == api.ProvisioningStateDeleting):
					s.b.DeleteDoc(doc)
				default:
					// TODO: improve memory usage by storing a subset of doc in mon.docs
					s.b.UpsertDoc(doc)
				}
			}

			s.mu.Unlock()
		}

		if successful {
			s.lastChangefeed.Store(time.Now())
		}

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}

// worker reads clusters to be monitored and monitors them
func (s *service) worker(stop <-chan struct{}, delay time.Duration, id string) {
	defer recover.Panic(s.baseLog)

	time.Sleep(delay)

	log := s.baseLog
	{
		s.mu.RLock()
		v := s.b.Doc(id)
		s.mu.RUnlock()

		if v == nil {
			return
		}

		log = utillog.EnrichWithResourceID(log, v.OpenShiftCluster.ID)
	}

	log.Debug("starting service")

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	h := time.Now().Hour()

out:
	for {
		s.mu.RLock()
		v := s.b.Doc(id)
		s.mu.RUnlock()

		if v == nil {
			break
		}

		newh := time.Now().Hour()

		s.workOne(context.Background(), log, v, newh != h)

		select {
		case <-t.C:
		case <-stop:
			break out
		}

		h = newh
	}

	log.Debug("stopping actuator")
}

// workOne checks the API server health of a cluster
func (s *service) workOne(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, hourlyRun bool) {
	restConfig, err := restconfig.RestConfig(s.dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return
	}

	m, err := NewActuator(ctx, s.env, log, restConfig, s.dbOpenShiftClusters, s.dbMaintenanceManifests)
	if err != nil {
		log.Error(err)
		s.m.EmitGauge("actuator.cluster.failedworker", 1, map[string]string{
			"resourceId": doc.OpenShiftCluster.ID,
		})
		return
	}

	_, err = m.Process(ctx, doc)
	if err != nil {
		log.Error(err)
	}
}
