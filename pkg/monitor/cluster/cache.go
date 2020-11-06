package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) getClusterVersion(ctx context.Context) (*configv1.ClusterVersion, error) {
	if mon.cache.cv != nil {
		return mon.cache.cv, nil
	}

	var err error
	mon.cache.cv, err = mon.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	return mon.cache.cv, err
}

func (mon *Monitor) listClusterOperators(ctx context.Context) (*configv1.ClusterOperatorList, error) {
	if mon.cache.cos != nil {
		return mon.cache.cos, nil
	}

	var err error
	mon.cache.cos, err = mon.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	return mon.cache.cos, err
}

func (mon *Monitor) listNodes(ctx context.Context) (*v1.NodeList, error) {
	if mon.cache.ns != nil {
		return mon.cache.ns, nil
	}

	var err error
	mon.cache.ns, err = mon.cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	return mon.cache.ns, err
}

func (mon *Monitor) listDeployments(ctx context.Context) (*appsv1.DeploymentList, error) {
	if mon.cache.dl != nil {
		return mon.cache.dl, nil
	}

	var err error
	mon.cache.dl, err = mon.cli.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	return mon.cache.dl, err
}
