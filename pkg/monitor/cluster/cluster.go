package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Azure/go-autorest/autorest/azure"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/emitter"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const MONITOR_GOROUTINES_PER_CLUSTER = 5

var _ monitoring.Monitor = (*Monitor)(nil)

type collectorFunc func(context.Context) error

type Monitor struct {
	collectors []collectorFunc

	log       *logrus.Entry
	hourlyRun bool

	oc   *api.OpenShiftCluster
	dims map[string]string

	restconfig  *rest.Config
	cli         kubernetes.Interface
	configcli   configclient.Interface
	operatorcli operatorclient.Interface
	m           metrics.Emitter
	arocli      aroclient.Interface
	env         env.Interface
	rawClient   rest.Interface
	tenantID    string
	now         func() time.Time

	ocpclientset clienthelper.Interface

	// Namespaces that are OpenShift or ARO managed that we want to monitor
	namespacesToMonitor []string

	// OpenShift version of the cluster being monitored
	clusterDesiredVersion version.Version
	clusterActualVersion  version.Version

	// Limit for items per pagination query
	queryLimit int
	// Limit for goroutines per cluster Monitor instance
	parallelism int
}

func NewMonitor(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, env env.Interface, tenantID string, m metrics.Emitter, hourlyRun bool) (monitoring.Monitor, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	dims := map[string]string{
		dimension.ResourceID:           oc.ID,
		dimension.SubscriptionID:       r.SubscriptionID,
		dimension.ClusterResourceGroup: r.ResourceGroup,
		dimension.ResourceName:         r.ResourceName,
	}

	// configure the shared rest clients
	configShallowCopy := *restConfig
	configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()

	// share the transport between all clients
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	// set up the raw rest client that we use for healthz scraping
	configShallowCopyRaw := *restConfig
	configShallowCopyRaw.GroupVersion = &schema.GroupVersion{}
	configShallowCopyRaw.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	configShallowCopyRaw.UserAgent = rest.DefaultKubernetesUserAgent()
	rawClient, err := rest.RESTClientForConfigAndClient(&configShallowCopyRaw, httpClient)
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfigAndClient(restConfig, httpClient)
	if err != nil {
		return nil, err
	}

	configcli, err := configclient.NewForConfigAndClient(restConfig, httpClient)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	operatorcli, err := operatorclient.NewForConfigAndClient(restConfig, httpClient)
	if err != nil {
		return nil, err
	}

	// lazy discovery will not attempt to reach out to the apiserver immediately
	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, err
	}

	ocpclientset, err := client.New(restConfig, client.Options{
		Mapper: mapper,
	})
	if err != nil {
		return nil, err
	}

	mon := &Monitor{
		log:       log,
		hourlyRun: hourlyRun,

		oc:   oc,
		dims: dims,

		restconfig:  restConfig,
		cli:         cli,
		configcli:   configcli,
		operatorcli: operatorcli,
		arocli:      arocli,
		rawClient:   rawClient,
		now:         time.Now,

		env:                 env,
		tenantID:            tenantID,
		m:                   m,
		ocpclientset:        clienthelper.NewWithClient(log, ocpclientset),
		namespacesToMonitor: []string{},
		queryLimit:          50,
		parallelism:         MONITOR_GOROUTINES_PER_CLUSTER,
	}
	mon.collectors = []collectorFunc{
		mon.emitAroOperatorHeartbeat,
		mon.emitAroOperatorConditions,
		mon.emitNSGReconciliation,
		mon.emitClusterOperatorConditions,
		mon.emitClusterOperatorVersions,
		mon.emitClusterVersionConditions,
		mon.emitClusterVersions,
		mon.emitDaemonsetStatuses,
		mon.emitDeploymentStatuses,
		mon.emitMachineConfigPoolConditions,
		mon.emitMachineConfigPoolUnmanagedNodeCounts,
		mon.emitMachineConditions,
		mon.emitNodeConditions,
		mon.emitPodConditions,
		mon.emitReplicasetStatuses,
		mon.emitStatefulsetStatuses,
		mon.emitJobConditions,
		mon.emitCNVVirtualMachineInstanceStatuses,
		mon.emitSummary,
		mon.emitOperatorFlagsAndSupportBanner,
		mon.emitMaintenanceState,
		mon.emitMDSDCertificateExpiry,
		mon.emitIngressAndAPIServerCertificateExpiry,
		mon.emitEtcdCertificateExpiry,
		mon.emitPrometheusAlerts, // at the end for now because it's the slowest/least reliable
		mon.emitCWPStatus,
		mon.emitClusterAuthenticationType,
		mon.emitNetworkMTU,
	}

	return mon, nil
}

func (mon *Monitor) timeCall(ctx context.Context, f func(context.Context) error) (err error) {
	collectorName := steps.ShortName(f)

	// Don't run collectors if we have already timed out
	if ctx.Err() != nil {
		mon.log.Debugf("skipping %s because %s", collectorName, ctx.Err())
		mon.emitMonitorCollectorSkipped(collectorName)
		return &failureToRunClusterCollector{collectorName: collectorName, inner: ctx.Err()}
	}

	innerNow := mon.now()
	mon.log.Debugf("running %s", collectorName)

	// If the collector panics we should return the error (so that it bubbles
	// up) but not prevent any other collector from running.
	defer func() {
		if e := recover(); e != nil {
			err = &failureToRunClusterCollector{collectorName: collectorName, inner: &collectorPanic{panicValue: e}}
			mon.emitMonitorCollectorError(collectorName)
		}
	}()

	innerErr := f(ctx)
	if innerErr != nil {
		// emit metrics collection failures and collect the err, but
		// don't stop running other metric collections
		mon.emitMonitorCollectorError(collectorName)
		return &failureToRunClusterCollector{collectorName: collectorName, inner: innerErr}
	} else {
		timeToComplete := mon.now().Sub(innerNow).Seconds()
		mon.emitMonitorCollectionTiming(collectorName, timeToComplete)
		mon.log.Debugf("successfully ran cluster collector '%s' in %2f sec", collectorName, timeToComplete)
	}
	return nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) (_err error) {
	// guard for any monitor-level panics
	defer func() {
		if e := recover(); e != nil {
			_err = &monitoring.MonitorPanic{PanicValue: e}
		}
	}()

	errs := []error{}

	monitoringStartTime := mon.now()
	mon.log.Debug("monitoring")

	if mon.hourlyRun {
		mon.emitGauge("cluster.provisioning", 1, map[string]string{
			"provisioningState":       mon.oc.Properties.ProvisioningState.String(),
			"failedProvisioningState": mon.oc.Properties.FailedProvisioningState.String(),
		})
	}

	// This API server healthz check must be first, our geneva monitor relies on this metric to always be emitted.
	err := mon.timeCall(ctx, mon.emitAPIServerHealthzCode)
	if err != nil {
		errs = append(errs, err)

		// If API is not returning 200, fallback to checking ping and short circuit the rest of the checks
		err := mon.timeCall(ctx, mon.emitAPIServerPingCode)
		if err != nil {
			errs = append(errs, err)
		}

		return errors.Join(errs...)
	}

	err = mon.timeCall(ctx, mon.prefetchClusterVersion)
	if err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}

	// Determine the list of OpenShift (or ARO) managed namespaces that we will
	// query for -- this needs to succeed
	err = mon.timeCall(ctx, mon.fetchManagedNamespaces)
	if err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}

	// Run up to mon.parallelism (default: MONITOR_GOROUTINES_PER_CLUSTER)
	// goroutines for collecting metrics
	wg := new(errgroup.Group)
	wg.SetLimit(mon.parallelism)

	// Create a channel capable of buffering one error from every collector
	errChan := make(chan error, len(mon.collectors))

	for _, f := range mon.collectors {
		wg.Go(func() error {
			innerErr := mon.timeCall(ctx, f)
			if innerErr != nil {
				// NOTE: The channel only has room to accommodate one error per
				// collector, so if a collector needs to return multiple errors
				// they should be joined into a single one (see errors.Join)
				// before being added.
				errChan <- innerErr
			}
			return nil
		})
	}

	err = wg.Wait()
	if err != nil {
		errs = append(errs, err)
	}
	// collect up the errors in the buffered channel
	close(errChan)
	for e := range errChan {
		errs = append(errs, e)
	}

	// emit a metric with how long we took when we have no errors
	if len(errs) == 0 && ctx.Err() == nil {
		mon.emitFloat("monitor.cluster.duration", mon.now().Sub(monitoringStartTime).Seconds(), map[string]string{})
	}

	return errors.Join(errs...)
}

func (mon *Monitor) emitMonitorCollectorError(collectorName string) {
	emitter.EmitGauge(mon.m, "monitor.cluster.collector.error", 1, mon.dims, map[string]string{"collector": collectorName})
}

func (mon *Monitor) emitMonitorCollectorSkipped(collectorName string) {
	emitter.EmitGauge(mon.m, "monitor.cluster.collector.skipped", 1, mon.dims, map[string]string{"collector": collectorName})
}

func (mon *Monitor) emitMonitorCollectionTiming(collectorName string, duration float64) {
	emitter.EmitFloat(mon.m, "monitor.cluster.collector.duration", duration, mon.dims, map[string]string{"collector": collectorName})
}

func (mon *Monitor) emitGauge(m string, value int64, dims map[string]string) {
	emitter.EmitGauge(mon.m, m, value, mon.dims, dims)
}

func (mon *Monitor) emitFloat(m string, value float64, dims map[string]string) {
	emitter.EmitFloat(mon.m, m, value, mon.dims, dims)
}

func (m *Monitor) MonitorName() string {
	return "cluster"
}
