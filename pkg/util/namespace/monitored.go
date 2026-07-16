package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// MonitoredNamespaces is the curated set of OpenShift/ARO-managed namespaces that the cluster monitor queries
// when emitting pod.conditions, pod.containerstatuses, and pod.restartcounter metrics.
var MonitoredNamespaces = []string{
	"openshift-apiserver",
	"openshift-azure-logging",
	"openshift-azure-operator",
	"openshift-etcd",
	"openshift-ingress",
	"openshift-kube-apiserver",
	"openshift-machine-config-operator",
	"openshift-machine-api",
	"openshift-monitoring",
}
