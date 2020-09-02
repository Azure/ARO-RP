package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) getClusterVersion() (*configv1.ClusterVersion, error) {
	if mon.cache.cv != nil {
		return mon.cache.cv, nil
	}

	var err error
	mon.cache.cv, err = mon.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	return mon.cache.cv, err
}

func (mon *Monitor) listClusterOperators() (*configv1.ClusterOperatorList, error) {
	if mon.cache.cos != nil {
		return mon.cache.cos, nil
	}

	var err error
	mon.cache.cos, err = mon.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	return mon.cache.cos, err
}

func (mon *Monitor) listNodes() (*v1.NodeList, error) {
	if mon.cache.ns != nil {
		return mon.cache.ns, nil
	}

	var err error
	mon.cache.ns, err = mon.cli.CoreV1().Nodes().List(metav1.ListOptions{})
	return mon.cache.ns, err
}
