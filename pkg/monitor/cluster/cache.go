package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

// Anything that caches a List is an anti-pattern because of the potential
// memory usage.  Don't add caches here: work to remove them.

func (mon *Monitor) getClusterVersion(ctx context.Context) (*configv1.ClusterVersion, error) {
	if mon.cache.cv != nil {
		return mon.cache.cv, nil
	}

	var err error
	mon.cache.cv, err = mon.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	return mon.cache.cv, err
}

// TODO: remove this function and paginate
func (mon *Monitor) listClusterOperators(ctx context.Context) (*configv1.ClusterOperatorList, error) {
	if mon.cache.cos != nil {
		return mon.cache.cos, nil
	}

	var err error
	mon.cache.cos, err = mon.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	return mon.cache.cos, err
}

// TODO: remove this function and paginate
func (mon *Monitor) listNodes(ctx context.Context) (*corev1.NodeList, error) {
	if mon.cache.ns != nil {
		return mon.cache.ns, nil
	}

	var err error
	mon.cache.ns, err = mon.cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	return mon.cache.ns, err
}

// TODO: remove this function and paginate
func (mon *Monitor) listARODeployments(ctx context.Context) (*appsv1.DeploymentList, error) {
	if mon.cache.arodl != nil {
		return mon.cache.arodl, nil
	}

	var err error
	mon.cache.arodl, err = mon.cli.AppsV1().Deployments(pkgoperator.Namespace).List(ctx, metav1.ListOptions{})
	return mon.cache.arodl, err
}

func (mon *Monitor) listAROClusters(ctx context.Context) (*arov1alpha1.ClusterList, error) {
	if mon.cache.cs != nil {
		return mon.cache.cs, nil
	}

	var err error
	mon.cache.cs, err = mon.arocli.AroV1alpha1().Clusters().List(ctx, metav1.ListOptions{})
	return mon.cache.cs, err
}
