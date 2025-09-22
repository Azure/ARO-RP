package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type failureToRunHiveCollector struct {
	collectorName string
	inner         error
}

func (e *failureToRunHiveCollector) Error() string {
	if e.inner != nil {
		return fmt.Sprintf("failure running Hive collector '%s':\n%s", e.collectorName, stringutils.IndentLines(e.inner.Error(), "  "))
	}
	return fmt.Sprintf("failure running Hive collector '%s': <missing>", e.collectorName)
}

func (e *failureToRunHiveCollector) Is(err error) bool {
	errCollector, ok := err.(*failureToRunHiveCollector)
	if !ok {
		return false
	}

	return errCollector.collectorName == e.collectorName
}

func (e *failureToRunHiveCollector) Unwrap() error {
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
