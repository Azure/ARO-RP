package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var errFetchClusterVersion = errors.New("error fetching ClusterVersion")
var errFetchAROOperatorMasterDeployment = errors.New("error fetching ARO Operator master deployment")
var errListAROOperatorDeployments = errors.New("error listing ARO Operator deployments")
var errListReplicaSets = errors.New("error listing replicasets")
var errListNamespaces = errors.New("error listing cluster namespaces")

var errAPIServerHealthzFailure = errors.New("error fetching APIServer healthz endpoint")
var errAPIServerPingFailure = errors.New("error fetching APIServer healthz ping endpoint")

type failureToRunClusterCollector struct {
	collectorName string
	inner         error
}

func (e *failureToRunClusterCollector) Error() string {
	if e.inner != nil {
		return fmt.Sprintf("failure running cluster collector '%s':\n%s", e.collectorName, stringutils.IndentLines(e.inner.Error(), "  "))
	}
	return fmt.Sprintf("failure running cluster collector '%s': <missing>", e.collectorName)
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

type collectorPanic struct {
	panicValue any
}

func (e *collectorPanic) Error() string {
	return fmt.Sprintf("panic: '%v'", e.panicValue)
}

func (e *collectorPanic) Is(err error) bool {
	_, ok := err.(*collectorPanic)
	return ok
}
