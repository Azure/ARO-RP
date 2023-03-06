package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

type Monitor struct {
	log       *logrus.Entry
	hourlyRun bool

	oc   *api.OpenShiftCluster
	dims map[string]string

	restconfig *rest.Config
	cli        kubernetes.Interface
	configcli  configclient.Interface
	maocli     machineclient.Interface
	mcocli     mcoclient.Interface
	m          metrics.Emitter
	arocli     aroclient.Interface

	hiveclientset hiveclient.Interface

	// access below only via the helper functions in cache.go
	cache struct {
		cos   *configv1.ClusterOperatorList
		cs    *arov1alpha1.ClusterList
		cv    *configv1.ClusterVersion
		ns    *corev1.NodeList
		arodl *appsv1.DeploymentList
	}
}

func NewMonitor(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster, m metrics.Emitter, hiveRestConfig *rest.Config, hourlyRun bool) (*Monitor, error) {
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

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	configcli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	maocli, err := machineclient.NewForConfig(restConfig)
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

	var hiveclientset hiveclient.Interface
	if hiveRestConfig != nil {
		var err error
		hiveclientset, err = hiveclient.NewForConfig(hiveRestConfig)
		if err != nil {
			// TODO(hive): Update to fail once we have Hive everywhere in prod and dev
			log.Error(err)
		}
	}

	return &Monitor{
		log:       log,
		hourlyRun: hourlyRun,

		oc:   oc,
		dims: dims,

		restconfig:    restConfig,
		cli:           cli,
		configcli:     configcli,
		maocli:        maocli,
		mcocli:        mcocli,
		arocli:        arocli,
		m:             m,
		hiveclientset: hiveclientset,
	}, nil
}

// Monitor checks the API server health of a cluster
func (mon *Monitor) Monitor(ctx context.Context) (errs []error) {
	mon.log.Debug("monitoring")

	if mon.hourlyRun {
		mon.emitGauge("cluster.provisioning", 1, map[string]string{
			"provisioningState":       mon.oc.Properties.ProvisioningState.String(),
			"failedProvisioningState": mon.oc.Properties.FailedProvisioningState.String(),
		})
	}

	// If API is not returning 200, don't need to run the next checks
	statusCode, err := mon.emitAPIServerHealthzCode(ctx)
	if err != nil {
		errs = append(errs, err)
		friendlyFuncName := steps.FriendlyName(mon.emitAPIServerHealthzCode)
		mon.log.Printf("%s: %s", friendlyFuncName, err)
		mon.emitGauge("monitor.clustererrors", 1, map[string]string{"monitor": friendlyFuncName})
	}
	if statusCode != http.StatusOK {
		return
	}
	for _, f := range []func(context.Context) error{
		mon.emitAroOperatorHeartbeat,
		mon.emitClusterObjectConditions,
		mon.emitClusterOperatorConditions,
		mon.emitClusterOperatorVersions,
		mon.emitClusterVersionConditions,
		mon.emitClusterVersions,
		mon.emitDaemonsetStatuses,
		mon.emitDeploymentStatuses,
		mon.emitMachineConfigPoolConditions,
		mon.emitMachineConfigPoolUnmanagedNodeCounts,
		mon.emitNodeConditions,
		mon.emitPodConditions,
		mon.emitReplicasetStatuses,
		mon.emitStatefulsetStatuses,
		mon.emitJobConditions,
		mon.emitSummary,
		mon.emitHiveRegistrationStatus,
		mon.emitOperatorFlagsAndSupportBanner,
		mon.emitPrometheusAlerts, // at the end for now because it's the slowest/least reliable
	} {
		err = f(ctx)
		if err != nil {
			errs = append(errs, err)
			friendlyFuncName := steps.FriendlyName(f)
			mon.log.Printf("%s: %s", friendlyFuncName, err)
			mon.emitGauge("monitor.clustererrors", 1, map[string]string{"monitor": friendlyFuncName})
			// keep going
		}
	}

	return
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
