package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "fmt"

type MonitorPanic struct {
	PanicValue any
}

func (e *MonitorPanic) Error() string {
	return fmt.Sprintf("monitor panic: '%v'", e.PanicValue)
}

func (e *MonitorPanic) Is(err error) bool {
	_, ok := err.(*MonitorPanic)
	return ok
}
