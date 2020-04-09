package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// TODO: After OpenShift 4.4, replace github.com/openshift/cluster-api with github.com/openshift/machine-api-operator
import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// OpenShiftClusterEnricher must update the cluster object with
// data received from API server
type OpenShiftClusterEnricher interface {
	Enrich(ctx context.Context, docs ...*api.OpenShiftClusterDocument)
}

// OpenShiftClusterPersistingEnricher must update the cluster object with
// data received from API server and be able to persist data into the DB
type OpenShiftClusterPersistingEnricher interface {
	OpenShiftClusterEnricher
	EnrichAndPersist(ctx context.Context, docs ...*api.OpenShiftClusterDocument)
}

type enricherTaskConstructor func(*logrus.Entry, *rest.Config, *api.OpenShiftCluster) (enricherTask, error)
type enricherTask interface {
	SetDefaults()
	FetchData(chan<- func(), chan<- error)
}

// NewBestEffortEnricher returns an enricher that attempts to populate
// fields, but ignores errors in case of failures
func NewBestEffortEnricher(log *logrus.Entry, env env.Interface, m metrics.Interface) OpenShiftClusterEnricher {
	return &bestEffortEnricher{
		log: log,
		env: env,
		m:   m,

		restConfig: restconfig.RestConfig,
		taskConstructors: []enricherTaskConstructor{
			newClusterVersionEnricherTask,
			newWorkerProfilesEnricherTask,
		},
	}
}

// NewCachingEnricher updates documents in the DB after inner enricher finished
func NewCachingEnricher(log *logrus.Entry, m metrics.Interface, db *database.Database, inner OpenShiftClusterEnricher, cacheTTL time.Duration) OpenShiftClusterPersistingEnricher {
	return &cachingEnricher{
		now:      time.Now,
		log:      log,
		m:        m,
		db:       db,
		inner:    inner,
		cacheTTL: cacheTTL,
	}
}

type bestEffortEnricher struct {
	log *logrus.Entry
	env env.Interface
	m   metrics.Interface

	restConfig       func(env env.Interface, oc *api.OpenShiftCluster) (*rest.Config, error)
	taskConstructors []enricherTaskConstructor
}

func (e *bestEffortEnricher) Enrich(ctx context.Context, docs ...*api.OpenShiftClusterDocument) {
	e.m.EmitGauge("enricher.tasks.count", int64(len(e.taskConstructors)*len(docs)), nil)

	var wg sync.WaitGroup
	wg.Add(len(docs))
	for i := range docs {
		go func(i int) {
			defer wg.Done()
			e.enrichOne(ctx, docs[i])
		}(i) // https://golang.org/doc/faq#closures_and_goroutines
	}
	wg.Wait()
}

func (e *bestEffortEnricher) enrichOne(ctx context.Context, doc *api.OpenShiftClusterDocument) {
	if !e.isValidProvisioningState(doc.OpenShiftCluster) {
		return
	}

	restConfig, err := e.restConfig(e.env, doc.OpenShiftCluster)
	if err != nil {
		e.m.EmitGauge("enricher.tasks.errors", int64(len(e.taskConstructors)), nil)
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
		task, err := e.taskConstructors[i](e.log, restConfig, doc.OpenShiftCluster)
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
			t := time.Now()
			defer func() {
				e.m.EmitGauge("enricher.tasks.duration", time.Now().Sub(t).Milliseconds(), map[string]string{
					"task": fmt.Sprintf("%T", tasks[i]),
				})
			}()

			tasks[i].FetchData(callbacks, errors)
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
			e.m.EmitGauge("enricher.tasks.timeouts", 1, nil)
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}

type cachingEnricher struct {
	now      func() time.Time
	log      *logrus.Entry
	db       *database.Database
	m        metrics.Interface
	inner    OpenShiftClusterEnricher
	cacheTTL time.Duration
}

func (e *cachingEnricher) Enrich(ctx context.Context, docs ...*api.OpenShiftClusterDocument) {
	e.enrich(ctx, docs...)
}

func (e *cachingEnricher) enrich(ctx context.Context, docs ...*api.OpenShiftClusterDocument) []*api.OpenShiftClusterDocument {
	now := e.now().UTC()
	filteredDocs := make([]*api.OpenShiftClusterDocument, 0, len(docs))

	for _, doc := range docs {
		if doc.LastEnrichment == nil || now.Sub(*doc.LastEnrichment) > e.cacheTTL {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	missCount := len(filteredDocs)
	hitCount := len(docs) - len(filteredDocs)

	e.m.EmitGauge("enricher.cache.hit.count", int64(hitCount), nil)
	e.m.EmitGauge("enricher.cache.miss.count", int64(missCount), nil)

	if missCount == 0 {
		return nil
	}

	e.inner.Enrich(ctx, filteredDocs...)

	for _, doc := range filteredDocs {
		doc.LastEnrichment = &now
	}

	return filteredDocs
}

func (e *cachingEnricher) EnrichAndPersist(ctx context.Context, docs ...*api.OpenShiftClusterDocument) {
	updatedDocs := e.enrich(ctx, docs...)

	// Ideally we should perform bulk update here, but there is no easy and reliable way to do it.
	// .NET and Java Cosmos DB libraries include support of bulk operations [1],
	// but the implementation is very complex and it seems to rely on undocumented system stored procedures.
	// [1] https://docs.microsoft.com/en-us/azure/cosmos-db/bulk-executor-overview
	for _, doc := range updatedDocs {
		_, err := e.db.OpenShiftClusters.Update(ctx, doc)
		if err != nil {
			e.log.Error(err)
			e.m.EmitGauge("enricher.cache.miss.errors", 1, nil)
		}
	}
}
