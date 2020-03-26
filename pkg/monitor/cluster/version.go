package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterVersion() error {
	restConfig, err := restconfig.RestConfig(mon.env, mon.oc)
	if err != nil {
		return err
	}

	configCli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	ver, err := configCli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		return err
	}

	mon.emitGauge("cluster.version", 1, map[string]string{
		"version": ver.Status.Desired.Version,
	})

	return err
}
