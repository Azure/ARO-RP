package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// FriendlyName returns a "friendly" stringified name of the given func.
func FriendlyName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// Step is the interface for steps that Runner can execute.
type Step interface {
	run(ctx context.Context, log *logrus.Entry) error
	String() string
}

// Run executes the provided steps in order until one fails or all steps
// are completed. Errors from failed steps are returned directly.
func Run(ctx context.Context, log *logrus.Entry, pollInterval time.Duration, steps []Step) error {
	for _, step := range steps {
		log.Infof("running step %s", step)
		err := step.run(ctx, log)

		if err != nil {
			log.Errorf("step %s encountered error: %s", step, err.Error())
			return err
		}
	}
	return nil
}
