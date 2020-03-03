package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// MetricAPIServerHeatlth is the metric name for the api server health
	MetricAPIServerHeatlth string = "ApiServerCode"
	// MetricNodesStatus is the metric name for the nodes status
	MetricNodesStatus string = "NodeNotReady"
	// MetricPrometheusAlert is the metric name for the prometheus alerts fired
	MetricPrometheusAlert string = "PrometheusAlert"
)

const (
	// AlertNamespace is the namespace where the alert manager pod is living
	AlertNamespace string = "openshift-monitoring"
	// AlertPodPrefix is the pod name prefix to query
	AlertPodPrefix string = "alertmanager-main"
	// AlertServiceEndpoint is the service name to query
	AlertServiceEndpoint string = "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts"
)
