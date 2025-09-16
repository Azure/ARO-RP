package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
)

var errFetchClusterVersion = errors.New("error fetching ClusterVersion")
var errFetchAROOperatorMasterDeployment = errors.New("error fetching ARO Operator master deployment")
var errListAROOperatorDeployments = errors.New("error listing ARO Operator deployments")
var errListReplicaSets = errors.New("error listing replicasets")
var errListNamespaces = errors.New("error")

var errAPIServerHealthzFailure = errors.New("error fetching APIServer healthz endpoint")
var errAPIServerPingFailure = errors.New("error fetching APIServer healthz ping endpoint")

type failureToRunClusterCollector struct {
	collectorName string
	inner         error
}

func (e *failureToRunClusterCollector) Error() string {
	return fmt.Sprintf("failure running cluster collector '%s'", e.collectorName)
}

func (e *failureToRunClusterCollector) Is(err error) bool {
	errCollector, ok := err.(*failureToRunClusterCollector)
	if !ok {
		return false
	}

	return errCollector.collectorName == e.collectorName
}

func (e *failureToRunClusterCollector) Unwrap() error {
	return e.inner
}
