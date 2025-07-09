package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type BestEffortEnricher interface {
	Enrich(ctx context.Context, log *logrus.Entry, ocs ...*api.OpenShiftCluster)
}

// ParallelEnricher enriches the cluster in parallel and does not fail if any of the
// enricher fails
type ParallelEnricher struct {
	enrichers map[string]ClusterEnricher
	emitter   metrics.Emitter
	dialer    proxy.Dialer
	//only used for testing because the metrics emission is async
	metricsWG *sync.WaitGroup
}

type ClusterEnricher interface {
	Enrich(context.Context, *logrus.Entry, *api.OpenShiftCluster, kubernetes.Interface, configclient.Interface, machineclient.Interface, operatorclient.Interface) error
	SetDefaults(*api.OpenShiftCluster)
}

type clients struct {
	k8s      kubernetes.Interface
	config   configclient.Interface
	machine  machineclient.Interface
	operator operatorclient.Interface
}

const (
	servicePrincipal = "servicePrincipal"
	ingressProfile   = "ingressProfile"
	clusterVersion   = "clusterVersion"
	machineClient    = "machineClient"
)

func NewParallelEnricher(metricsEmitter metrics.Emitter, dialer proxy.Dialer) ParallelEnricher {
	return ParallelEnricher{
		emitter: metricsEmitter,
		enrichers: map[string]ClusterEnricher{
			servicePrincipal: clusterServicePrincipalEnricher{},
			ingressProfile:   ingressProfileEnricher{},
			clusterVersion:   clusterVersionEnricher{},
			machineClient:    machineClientEnricher{},
		},
		dialer: dialer,
	}
}

func (p ParallelEnricher) Enrich(ctx context.Context, log *logrus.Entry, ocs ...*api.OpenShiftCluster) {
	var wg sync.WaitGroup
	wg.Add(len(ocs))
	for _, oc := range ocs {
		// https://golang.org/doc/faq#closures_and_goroutines
		oc := oc
		go func() {
			defer recover.Panic(log)
			defer wg.Done()

			k8sclient, machineclient, operatorclient, configclient, unsuccessfulEnrichers := p.initializeClients(ctx, log, oc)
			clients := clients{
				k8s:      k8sclient,
				machine:  machineclient,
				operator: operatorclient,
				config:   configclient,
			}

			p.enrichOne(ctx, log, oc, clients, unsuccessfulEnrichers)
		}()
	}
	wg.Wait()
}

func (p ParallelEnricher) enrichOne(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster, clients clients, unsuccessfulEnrichers map[string]bool) {
	if err := validateProvisioningState(log, oc); err != nil {
		log.Info(err)
		return
	}

	numEnrichers := len(p.enrichers)

	// We skip the service principal enricher for workload identity clusters
	if oc.UsesWorkloadIdentity() {
		numEnrichers = numEnrichers - 1
	}

	errors := make(chan error, numEnrichers)
	expectedResults := 0
	for name, enricher := range p.enrichers {
		if unsuccessfulEnrichers[name] || enricher == nil || (name == servicePrincipal && oc.UsesWorkloadIdentity()) {
			continue
		}
		p.emitter.EmitGauge("enricher.tasks.count", 1, nil)

		expectedResults++

		e := enricher
		go func() {
			t := time.Now()

			//only used in testing
			if p.metricsWG != nil {
				p.metricsWG.Add(1)
				defer p.metricsWG.Done()
			}

			e.SetDefaults(oc)
			errors <- e.Enrich(ctx, log, oc, clients.k8s, clients.config, clients.machine, clients.operator)

			p.emitter.EmitGauge(
				"enricher.tasks.duration",
				time.Since(t).Milliseconds(),
				map[string]string{"task": fmt.Sprintf("%T", e)})
		}()
	}

	p.waitForResults(log, errors, expectedResults)
}

func (p ParallelEnricher) waitForResults(log *logrus.Entry, errChannel chan error, expectedResults int) {
	timeout := false
	//retrieve the errors from the routines
	for i := 0; i < expectedResults; i++ {
		err := <-errChannel
		switch err {
		case nil:
			//do nothing
		case context.Canceled, context.DeadlineExceeded:
			if !timeout {
				p.emitter.EmitGauge("enricher.timeouts", 1, nil)
				log.Warn("timeout expired")
				timeout = true
			}
		default:
			p.taskError(log, err, 1)
		}
	}
}

// initializeClients initialize the necassary clients for the specified cluster
// if some clients fail to be initialized, it also returns the list of enrichers
// that we should skip because the clients they are using failed to instantiate
func (p ParallelEnricher) initializeClients(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster) (
	k8s kubernetes.Interface, machineclient machineclient.Interface, operatorclient operatorclient.Interface, configclient configclient.Interface, unsuccessfulEnrichers map[string]bool) {
	unsuccessfulEnrichers = make(map[string]bool)
	k8s, err := p.setupK8sClient(ctx, oc)
	if err != nil {
		unsuccessfulEnrichers[servicePrincipal] = true
		unsuccessfulEnrichers[ingressProfile] = true
		p.taskError(log, err, 2)
	}
	machineclient, err = p.setupMachineClient(ctx, oc)
	if err != nil {
		unsuccessfulEnrichers[machineClient] = true
		p.taskError(log, err, 1)
	}
	operatorclient, err = p.setupOperatorClient(ctx, oc)
	if err != nil {
		unsuccessfulEnrichers[ingressProfile] = true
		p.taskError(log, err, 1)
	}
	configclient, err = p.setupConfigClient(ctx, oc)
	if err != nil {
		unsuccessfulEnrichers[clusterVersion] = true
		p.taskError(log, err, 1)
	}
	return k8s, machineclient, operatorclient, configclient, unsuccessfulEnrichers
}

func (p ParallelEnricher) taskError(log *logrus.Entry, err error, count int) {
	p.emitter.EmitGauge("enricher.tasks.errors", int64(count), nil)
	log.Error(err)
}

func (p ParallelEnricher) setupK8sClient(ctx context.Context, oc *api.OpenShiftCluster) (kubernetes.Interface, error) {
	restConfig, err := restconfig.RestConfig(p.dialer, oc)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

func (p ParallelEnricher) setupConfigClient(ctx context.Context, oc *api.OpenShiftCluster) (configclient.Interface, error) {
	restConfig, err := restconfig.RestConfig(p.dialer, oc)
	if err != nil {
		return nil, err
	}

	return configclient.NewForConfig(restConfig)
}

func (p ParallelEnricher) setupOperatorClient(ctx context.Context, oc *api.OpenShiftCluster) (operatorclient.Interface, error) {
	restConfig, err := restconfig.RestConfig(p.dialer, oc)
	if err != nil {
		return nil, err
	}
	return operatorclient.NewForConfig(restConfig)
}

func (p ParallelEnricher) setupMachineClient(ctx context.Context, oc *api.OpenShiftCluster) (machineclient.Interface, error) {
	restConfig, err := restconfig.RestConfig(p.dialer, oc)
	if err != nil {
		return nil, err
	}

	return machineclient.NewForConfig(restConfig)
}

func validateProvisioningState(log *logrus.Entry, oc *api.OpenShiftCluster) error {
	switch oc.Properties.ProvisioningState {
	case api.ProvisioningStateCreating, api.ProvisioningStateDeleting:
		return fmt.Errorf("cluster is in %q provisioning state. Skipping enrichment", oc.Properties.ProvisioningState)
	case api.ProvisioningStateFailed:
		switch oc.Properties.FailedProvisioningState {
		case api.ProvisioningStateCreating, api.ProvisioningStateDeleting:
			return fmt.Errorf("cluster is in failed %q provisioning state. Skipping enrichment", oc.Properties.ProvisioningState)
		}
	}
	return nil
}
