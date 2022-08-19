package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// OpenShiftClusterEnricher must update the cluster object with
// data received from API server
type OpenShiftClusterEnricher interface {
	Enrich(ctx context.Context, ocs ...*api.OpenShiftCluster)
}

type enricherTaskConstructor func(*logrus.Entry, *rest.Config, *api.OpenShiftCluster) (enricherTask, error)
type enricherTask interface {
	SetDefaults()
	FetchData(context.Context, chan<- func(), chan<- error)
}

// NewBestEffortEnricher returns an enricher that attempts to populate
// fields, but ignores errors in case of failures
func NewBestEffortEnricher(log *logrus.Entry, m metrics.Emitter) OpenShiftClusterEnricher {
	return &bestEffortEnricher{
		log: log,
		m:   m,

		restConfig: restconfig.RestConfig,
		taskConstructors: []enricherTaskConstructor{
			newClusterVersionEnricherTask,
			newWorkerProfilesEnricherTask,
			newClusterServicePrincipalEnricherTask,
			newIngressProfileEnricherTask,
		},
	}
}

type bestEffortEnricher struct {
	log *logrus.Entry
	m   metrics.Emitter

	restConfig       func(oc *api.OpenShiftCluster) (*rest.Config, error)
	taskConstructors []enricherTaskConstructor
}

func (e *bestEffortEnricher) Enrich(ctx context.Context, ocs ...*api.OpenShiftCluster) {
	e.m.EmitGauge("enricher.tasks.count", int64(len(e.taskConstructors)*len(ocs)), nil)

	var wg sync.WaitGroup
	wg.Add(len(ocs))
	for i := range ocs {
		go func(i int) {
			defer recover.Panic(e.log)

			defer wg.Done()
			e.enrichOne(ctx, ocs[i])
		}(i) // https://golang.org/doc/faq#closures_and_goroutines
	}
	wg.Wait()
}

func (e *bestEffortEnricher) enrichOne(ctx context.Context, oc *api.OpenShiftCluster) {
	if !e.isValidProvisioningState(oc) {
		return
	}

	restConfig, err := e.restConfig(oc)
	if err != nil {
		e.m.EmitGauge("enricher.tasks.errors", int64(len(e.taskConstructors)), nil)
		e.log.Error(err)
		return
	}

	tasks := make([]enricherTask, 0, len(e.taskConstructors))
	for i := range e.taskConstructors {
		task, err := e.taskConstructors[i](e.log, restConfig, oc)
		if err != nil {
			e.m.EmitGauge("enricher.tasks.errors", 1, nil)
			e.log.Error(err)
			continue
		}
		tasks = append(tasks, task)
		task.SetDefaults()
	}

	// We must use a buffered channels with length equal to the number of tasks
	// to ensure that FetchData goroutines return as soon as they finish
	// their job and do not wait for someone to read from the channel.
	// Otherwise in case of error in one of the goroutines or on timeout
	// they will not be garbage collected.
	callbacks := make(chan func(), len(tasks))
	errors := make(chan error, len(tasks))
	for i := range tasks {
		go func(i int) {
			defer recover.Panic(e.log)

			t := time.Now()
			defer func() {
				e.m.EmitGauge("enricher.tasks.duration", time.Since(t).Milliseconds(), map[string]string{
					"task": fmt.Sprintf("%T", tasks[i]),
				})
			}()

			tasks[i].FetchData(ctx, callbacks, errors)
		}(i) // https://golang.org/doc/faq#closures_and_goroutines
	}

out:
	for i := 0; i < len(tasks); i++ {
		select {
		case f := <-callbacks:
			f()
		case <-errors:
			// No need to log errors. We log them in each task locally
			e.m.EmitGauge("enricher.tasks.errors", 1, nil)
		case <-ctx.Done():
			e.m.EmitGauge("enricher.timeouts", 1, nil)
			e.log.Warn("timeout expired")
			break out
		}
	}
}

// isValidProvisioningState checks whether or not it is ok to run enrichment
// of the object based on the ProvisioningState.
// For example, when a user creates a new cluster kubeconfig for the cluster
// will be missing from the object in the beginning of the creation process
// and it will be not possible to make requests to the API server.
func (e *bestEffortEnricher) isValidProvisioningState(oc *api.OpenShiftCluster) bool {
	switch oc.Properties.ProvisioningState {
	case api.ProvisioningStateCreating, api.ProvisioningStateDeleting:
		e.log.Infof("cluster is in %q provisioning state. Skipping enrichment...", oc.Properties.ProvisioningState)
		return false
	case api.ProvisioningStateFailed:
		switch oc.Properties.FailedProvisioningState {
		case api.ProvisioningStateCreating, api.ProvisioningStateDeleting:
			e.log.Infof("cluster is in failed %q provisioning state. Skipping enrichment...", oc.Properties.ProvisioningState)
			return false
		}
	}
	return true
}
