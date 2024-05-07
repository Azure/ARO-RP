package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/go-autorest/autorest/to"
)

const maxDequeueCount = 5

func (s *service) try(ctx context.Context) (bool, error) {

	doc, err := s.dbMaintenanceManifests.Dequeue(ctx)
	if err != nil || doc == nil {
		return false, err
	}

	log := s.baseLog
	log = utillog.EnrichWithResourceID(log, doc.ClusterID)

	if doc.Dequeues > maxDequeueCount {
		err := fmt.Errorf("dequeued %d times, failing", doc.Dequeues)
		_, leaseErr := s.dbMaintenanceManifests.EndLease(ctx, doc.ClusterID, doc.ID, api.MaintenanceManifestStateTimedOut, to.StringPtr(err.Error()))
		return true, leaseErr
	}

	log.Print("dequeued")
	s.workers.Add(1)
	s.m.EmitGauge("mimo.actuator.workers.count", int64(s.workers.Load()), nil)

	go func() {
		defer recover.Panic(log)

		t := time.Now()

		defer func() {
			s.workers.Add(-1)
			s.m.EmitGauge("mimo.actuator.workers.count", int64(s.workers.Load()), nil)
			s.cond.Signal()

			log.WithField("duration", time.Since(t).Seconds()).Print("done")
		}()

		err := s.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}
	}()
	return true, nil
}

func (s *service) handle(ctx context.Context, log *logrus.Entry, doc *api.MaintenanceManifestDocument) error {
	// Get a lease on the OpenShiftClusterDocument
	var oc *api.OpenShiftClusterDocument
	var err error

	release := func() {
		if doc.LeaseOwner != "" {
			_, err = s.dbMaintenanceManifests.EndLease(ctx, doc.ClusterID, doc.ID, api.MaintenanceManifestStateTimedOut, nil)
			if err != nil {
				log.Error(err)
			}
		}

		if oc == nil || oc.LeaseOwner == "" {
			return
		}
		oc, err = s.dbOpenShiftClusters.EndLease(ctx, doc.ResourceID, oc.OpenShiftCluster.Properties.ProvisioningState, oc.OpenShiftCluster.Properties.LastProvisioningState, nil)
		if err != nil {
			log.Error(err)
		}
	}
	defer release()

	timeout, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// Wait a little bit to get the OpenShiftClusters lease
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var inerr error
		oc, inerr = s.dbOpenShiftClusters.Lease(ctx, doc.ClusterID)
		if inerr != nil {
			if inerr.Error() == "lost lease" {
				return false, nil
			}
			return false, inerr
		}

		return true, nil
	}, timeout.Done())

	restConfig, err := restconfig.RestConfig(s.dialer, oc.OpenShiftCluster)
	if err != nil {
		return err
	}

	actuator, err := NewActuator(ctx, s.env, log, restConfig, s.dbMaintenanceManifests)
	if err != nil {
		return err
	}

	_, err = actuator.Process(ctx, doc, oc)
	if err != nil {
		s.m.EmitGauge("actuator.cluster.failedworker", 1, map[string]string{
			"resourceId": doc.ClusterID,
		})
		return err
	}

	return nil

}
