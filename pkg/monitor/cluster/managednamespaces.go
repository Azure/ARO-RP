package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

var scopedNamespaces = []string{
	"openshift-apiserver",
	"openshift-apiserver-operator",
	"openshift-authentication",
	"openshift-authentication-operator",
	"openshift-azure-logging",
	"openshift-azure-operator",
	"openshift-dns",
	"openshift-dns-operator",
	"openshift-etcd",
	"openshift-etcd-operator",
	"openshift-ingress",
	"openshift-ingress-operator",
	"openshift-kube-apiserver",
	"openshift-kube-apiserver-operator",
	"openshift-kube-controller-manager",
	"openshift-kube-controller-manager-operator",
	"openshift-kube-scheduler",
	"openshift-kube-scheduler-operator",
	"openshift-machine-config-operator",
	"openshift-machine-api",
	"openshift-monitoring",
	"openshift-monitoring-operator",
	"openshift-network-operator",
	"openshift-ovn-kubernetes",
	"openshift-sdn",
}

func (mon *Monitor) fetchManagedNamespaces(ctx context.Context) error {
	mon.namespacesToMonitor = scopedNamespaces
	return nil
}
