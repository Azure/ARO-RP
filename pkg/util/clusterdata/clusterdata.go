package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// TODO: After OpenShift 4.4, replace github.com/openshift/cluster-api with github.com/openshift/machine-api-operator
import (
	"context"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
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
	FetchData(chan<- func(), chan<- error)
}

// NewBestEffortEnricher returns an enricher that attempts to populate
// fields, but ignores errors in case of failures
func NewBestEffortEnricher(log *logrus.Entry, env env.Interface) OpenShiftClusterEnricher {
	return &bestEffortEnricher{
		log: log,
		env: env,

		restConfig: restconfig.RestConfig,
		taskConstructors: []enricherTaskConstructor{
			newClusterVersionEnricherTask,
			newServicePrincipalEnricherTask,
			newWorkerProfilesEnricherTask,
		},
	}
}

type bestEffortEnricher struct {
	log *logrus.Entry
	env env.Interface

	restConfig       func(env env.Interface, oc *api.OpenShiftCluster) (*rest.Config, error)
	taskConstructors []enricherTaskConstructor
}

func (e *bestEffortEnricher) Enrich(ctx context.Context, ocs ...*api.OpenShiftCluster) {
	var wg sync.WaitGroup
	wg.Add(len(ocs))
	for i := range ocs {
		go func(i int) {
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

	restConfig, err := e.restConfig(e.env, oc)
	if err != nil {
		e.log.Error(err)
		return
	}

	// TODO: Get rid of the wrapping RoundTripper once implementation of the KEP below lands into openshift/client-go:
	//       https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20200123-client-go-ctx.md
	restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return rt.RoundTrip(req.WithContext(ctx))
		})
	})

	tasks := make([]enricherTask, 0, len(e.taskConstructors))
	for i := range e.taskConstructors {
		task, err := e.taskConstructors[i](e.log, restConfig, oc)
		if err != nil {
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
		go tasks[i].FetchData(callbacks, errors)
	}

out:
	for i := 0; i < len(tasks); i++ {
		select {
		case f := <-callbacks:
			f()
		case <-errors:
			// Ignore errors. We log them in each task locally
			continue
		case <-ctx.Done():
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
			e.log.Infof("cluster is in failed %q provisioning state. Skiping enrichment...", oc.Properties.ProvisioningState)
			return false
		}
	}
	return true
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
