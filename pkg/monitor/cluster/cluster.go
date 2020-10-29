package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"runtime"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type Monitor struct {
	dialer    proxy.Dialer
	log       *logrus.Entry
	hourlyRun bool

	oc   *api.OpenShiftCluster
	dims map[string]string

	cli       kubernetes.Interface
	configcli configclient.Interface
	mcocli    mcoclient.Interface
	m         metrics.Interface
	arocli    aroclient.AroV1alpha1Interface

	// access below only via the helper functions in cache.go
	cache struct {
		cos *configv1.ClusterOperatorList
		cv  *configv1.ClusterVersion
		ns  *v1.NodeList
		dl  *appsv1.DeploymentList
	}
}

func NewMonitor(ctx context.Context, dialer proxy.Dialer, log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Interface, hourlyRun bool) (*Monitor, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	dims := map[string]string{
		"resourceId":     oc.ID,
		"subscriptionId": r.SubscriptionID,
		"resourceGroup":  r.ResourceGroup,
		"resourceName":   r.ResourceName,
	}

	restConfig, err := restconfig.RestConfig(dialer, oc)
	if err != nil {
		return nil, err
	}

	// TODO: Get rid of the wrapping RoundTripper once implementation of the KEP below lands into openshift/kubernetes-client-go:
	//       https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20200123-client-go-ctx.md
	restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return rt.RoundTrip(req.WithContext(ctx))
		})
	})

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	configcli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	mcocli, err := mcoclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Monitor{
		dialer:    dialer,
		log:       log,
		hourlyRun: hourlyRun,

		oc:   oc,
		dims: dims,

		cli:       cli,
		configcli: configcli,
		mcocli:    mcocli,
		arocli:    arocli,
		m:         m,
	}, nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) {
	mon.log.Debug("monitoring")

	// If API is not returning 200, don't need to run the next checks
	statusCode, err := mon.emitAPIServerHealthzCode()
	if err != nil {
		mon.log.Printf("%s: %s", runtime.FuncForPC(reflect.ValueOf(mon.emitAPIServerHealthzCode).Pointer()).Name(), err)
		mon.emitGauge("monitor.clustererrors", 1, map[string]string{"monitor": runtime.FuncForPC(reflect.ValueOf(mon.emitAPIServerHealthzCode).Pointer()).Name()})
	}
	if statusCode != http.StatusOK {
		return
	}

	for _, f := range []func(context.Context) error{
		mon.emitAroOperatorHeartbeat,
		mon.emitAroOperatorConditions,
		mon.emitClusterOperatorConditions,
		mon.emitClusterOperatorVersions,
		mon.emitClusterVersionConditions,
		mon.emitClusterVersions,
		mon.emitDaemonsetStatuses,
		mon.emitDeploymentStatuses,
		mon.emitMachineConfigPoolConditions,
		mon.emitNodeConditions,
		mon.emitPodConditions,
		mon.emitReplicasetStatuses,
		mon.emitStatefulsetStatuses,
		mon.emitSummary,
		mon.emitPrometheusAlerts, // at the end for now because it's the slowest/least reliable
	} {
		err = f(ctx)
		if err != nil {
			mon.log.Printf("%s: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), err)
			mon.emitGauge("monitor.clustererrors", 1, map[string]string{"monitor": runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()})
			// keep going
		}
	}
}

func (mon *Monitor) emitFloat(m string, value float64, dims map[string]string) {
	if dims == nil {
		dims = map[string]string{}
	}
	for k, v := range mon.dims {
		dims[k] = v
	}
	mon.m.EmitFloat(m, value, dims)
}

func (mon *Monitor) emitGauge(m string, value int64, dims map[string]string) {
	if dims == nil {
		dims = map[string]string{}
	}
	for k, v := range mon.dims {
		dims[k] = v
	}
	mon.m.EmitGauge(m, value, dims)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
