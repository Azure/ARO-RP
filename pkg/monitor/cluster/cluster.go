package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"runtime"

	"github.com/Azure/go-autorest/autorest/azure"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type Monitor struct {
	env env.Interface
	log *logrus.Entry

	oc   *api.OpenShiftCluster
	dims map[string]string

	cli       kubernetes.Interface
	configcli configclient.Interface
	mcocli    mcoclient.Interface
	m         metrics.Interface
}

func NewMonitor(ctx context.Context, env env.Interface, log *logrus.Entry, oc *api.OpenShiftCluster, m metrics.Interface) (*Monitor, error) {
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

	restConfig, err := restconfig.RestConfig(env, oc)
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

	return &Monitor{
		env: env,
		log: log,

		oc:   oc,
		dims: dims,

		cli:       cli,
		configcli: configcli,
		mcocli:    mcocli,
		m:         m,
	}, nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) {
	mon.log.Debug("monitoring")

	// If API is not returning 200, don't need to run the next checks
	statusCode, err := mon.emitAPIServerHealthzCode()
	if err != nil {
		mon.log.Error(err)
		return
	}
	if statusCode != http.StatusOK {
		return
	}

	for _, f := range []func(ctx context.Context) error{
		mon.emitClusterOperatorsMetrics,
		mon.emitClusterVersionMetrics,
		mon.emitNodesMetrics,
		mon.emitPrometheusAlerts,
		mon.emitMachineConfigPoolMetrics,
	} {
		err = f(ctx)
		if err != nil {
			mon.log.Errorf("%s: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), err)
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
